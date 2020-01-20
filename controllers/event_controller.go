/*
Copyright 2020 The Event Reactor authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	"context"
	"regexp"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/summerwind/eventreactor/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var eventTypeKey = ".spec.trigger.type"

// EventReconciler reconciles a Event object
type EventReconciler struct {
	client.Client
	Log    logr.Logger
	Scheme *runtime.Scheme
}

// +kubebuilder:rbac:groups=eventreactor.summerwind.dev,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventreactor.summerwind.dev,resources=events/status,verbs=get;update;patch

func (r *EventReconciler) Reconcile(req ctrl.Request) (ctrl.Result, error) {
	ctx := context.Background()
	log := r.Log.WithValues("event", req.NamespacedName)

	var instance v1alpha1.Event
	err := r.Get(ctx, req.NamespacedName, &instance)
	if err != nil {
		log.Error(err, "Unable to fetch Event")
		return ctrl.Result{}, client.IgnoreNotFound(err)
	}

	if instance.Status.DispatchTime != nil {
		log.V(1).Info("Already dispatched")
		return ctrl.Result{}, nil
	}

	opts := []client.ListOption{
		client.InNamespace(instance.Namespace),
		client.MatchingFields{eventTypeKey: instance.Spec.Type},
	}

	var subscriptionList v1alpha1.SubscriptionList
	err = r.List(ctx, &subscriptionList, opts...)
	if err != nil {
		log.V(1).Info("Phase 1")
		return ctrl.Result{}, err
	}

	for _, sub := range subscriptionList.Items {
		var (
			err     error
			matched bool
		)

		if sub.Spec.Trigger.Type != instance.Spec.Type {
			log.V(1).Info("Event type mismatched", "subscription", sub.Name)
			continue
		}

		if sub.Spec.Trigger.MatchSource != "" {
			matched, err = regexp.MatchString(sub.Spec.Trigger.MatchSource, instance.Spec.Source)
			if err != nil {
				log.V(0).Info("Invalid event source pattern", "subscription", sub.Name)
				continue
			}
			if !matched {
				log.V(1).Info("Event source mismatched", "subscription", sub.Name)
				continue
			}
		}

		if sub.Spec.Trigger.MatchSubject != "" {
			matched, err = regexp.MatchString(sub.Spec.Trigger.MatchSubject, instance.Spec.Subject)
			if err != nil {
				log.V(0).Info("Invalid event subject pattern", "subscription", sub.Name)
				continue
			}
			if !matched {
				log.V(1).Info("Event subject mismatched", "subscription", sub.Name)
				continue
			}
		}

		for _, tmpl := range sub.Spec.ResourceTemplates {
			res := tmpl.DeepCopy()
			current := tmpl.DeepCopy()

			res.SetNamespace(sub.Namespace)
			if res.GetName() == "" {
				res.SetName(sub.Name)
			}

			resReq := types.NamespacedName{
				Name:      res.GetName(),
				Namespace: res.GetNamespace(),
			}

			err = r.Get(ctx, resReq, current)
			if err != nil {
				if errors.IsNotFound(err) {
					err = r.Create(ctx, res)
					if err != nil {
						log.V(1).Info("Phase 2")
						return reconcile.Result{}, err
					}
					log.Info("Create a new resource with template", "namespace", res.GetNamespace(), "name", res.GetName(), "gvk", res.GroupVersionKind())
				} else {
					return ctrl.Result{}, err
				}
			} else {
				err = r.Update(ctx, res)
				if err != nil {
					log.V(1).Info("Phase 3", "object", current)
					return reconcile.Result{}, err
				}
				log.Info("Update resource with template", "namespace", res.GetNamespace(), "name", res.GetName(), "gvk", res.GroupVersionKind())
			}
		}
	}

	now := metav1.Now()
	event := instance.DeepCopy()
	event.Status.DispatchTime = &now

	err = r.Update(ctx, event)
	if err != nil {
		log.V(1).Info("Phase 4")
		return ctrl.Result{}, err
	}

	log.Info("Successfully dispatched")

	return ctrl.Result{}, nil
}

func (r *EventReconciler) SetupWithManager(mgr ctrl.Manager) error {
	err := mgr.GetFieldIndexer().IndexField(&v1alpha1.Subscription{}, eventTypeKey, func(obj runtime.Object) []string {
		sub := obj.(*v1alpha1.Subscription)
		return []string{sub.Spec.Trigger.Type}
	})
	if err != nil {
		return err
	}

	return ctrl.NewControllerManagedBy(mgr).
		For(&v1alpha1.Event{}).
		Complete(r)
}
