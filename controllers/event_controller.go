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
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"text/template"

	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	v1alpha1 "github.com/summerwind/eventreactor/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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
		log.Error(err, "Failed to get event")
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
		log.Error(err, "Failed to get subscription list")
		return ctrl.Result{}, err
	}

	for _, sub := range subscriptionList.Items {
		var (
			err     error
			matched bool
		)

		subLog := log.WithValues("subscription", fmt.Sprintf("%s/%s", sub.Namespace, sub.Name))

		if sub.Spec.Trigger.Type != instance.Spec.Type {
			subLog.V(1).Info("Event type mismatched")
			continue
		}

		if sub.Spec.Trigger.MatchSource != "" {
			matched, err = regexp.MatchString(sub.Spec.Trigger.MatchSource, instance.Spec.Source)
			if err != nil {
				subLog.Info("Invalid event source pattern")
				continue
			}
			if !matched {
				subLog.V(1).Info("Event source mismatched")
				continue
			}
		}

		if sub.Spec.Trigger.MatchSubject != "" {
			matched, err = regexp.MatchString(sub.Spec.Trigger.MatchSubject, instance.Spec.Subject)
			if err != nil {
				subLog.Info("Invalid event subject pattern")
				continue
			}
			if !matched {
				subLog.Info("Event subject mismatched")
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

			resLog := subLog.WithValues("kind", res.GroupVersionKind().Kind, "name", fmt.Sprintf("%s/%s", res.GetNamespace(), res.GetName()))

			err = expandVars(res, &instance)
			if err != nil {
				resLog.Error(err, "Failed to expand variables")
			}

			key := types.NamespacedName{
				Name:      res.GetName(),
				Namespace: res.GetNamespace(),
			}

			err = r.Get(ctx, key, current)
			if err != nil {
				if errors.IsNotFound(err) {
					err = r.Create(ctx, res)
					if err != nil {
						resLog.Error(err, "Failed to create resource")
						return reconcile.Result{}, err
					}
					resLog.Info("Resource created")
				} else {
					return ctrl.Result{}, err
				}
			} else {
				err = r.Update(ctx, res)
				if err != nil {
					resLog.Error(err, "Failed to update resource")
					return reconcile.Result{}, err
				}
				resLog.Info("Resource updated")
			}
		}
	}

	now := metav1.Now()
	event := instance.DeepCopy()
	event.Status.DispatchTime = &now

	err = r.Update(ctx, event)
	if err != nil {
		log.Error(err, "Failed to update event")
		return ctrl.Result{}, err
	}

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

func expandVars(res *unstructured.Unstructured, ev *v1alpha1.Event) error {
	content := res.UnstructuredContent()

	resBytes, err := json.Marshal(content)
	if err != nil {
		return err
	}

	tmpl, err := template.New("resource").Delims("((", "))").Parse(string(resBytes))
	if err != nil {
		return err
	}

	vars := struct {
		Event *v1alpha1.Event
		Data  interface{}
	}{
		Event: ev,
		Data:  nil,
	}

	buf := bytes.NewBuffer([]byte{})
	if err := tmpl.Execute(buf, vars); err != nil {
		return err
	}

	if err := json.Unmarshal(buf.Bytes(), &content); err != nil {
		return err
	}

	res.SetUnstructuredContent(content)

	return nil
}
