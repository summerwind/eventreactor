/*
Copyright 2018 The Event Reactor Authors.

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

package event

import (
	"context"
	"fmt"
	"regexp"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/tools/record"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	ControllerName = "event-controller"
)

// Add creates a new Event Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEvent{
		Client:   mgr.GetClient(),
		scheme:   mgr.GetScheme(),
		recorder: mgr.GetRecorder(ControllerName),
		log:      logf.Log.WithName(ControllerName),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New(ControllerName, mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Event
	err = c.Watch(&source.Kind{Type: &v1alpha1.Event{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileEvent{}

// ReconcileEvent reconciles a Event object
type ReconcileEvent struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	log      logr.Logger
}

// Reconcile reads that state of the cluster for a Event object and makes changes based on the state read
// and what is in the Event.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=pipelines,verbs=get;list;watch
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=actions,verbs=get;list;watch;create
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
func (r *ReconcileEvent) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Event instance
	instance := &v1alpha1.Event{}
	err := r.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Object not found, return.  Created objects are automatically garbage collected.
			// For additional cleanup logic use finalizers.
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	if instance.Status.DispatchTime != nil {
		return reconcile.Result{}, nil
	}

	pipelineList := &v1alpha1.PipelineList{}

	labels := map[string]string{}
	labels[v1alpha1.KeyEventType] = instance.Spec.Type

	opts := &client.ListOptions{Namespace: instance.Namespace}
	opts = opts.MatchingLabels(labels)

	err = r.List(context.TODO(), opts, pipelineList)
	if err != nil {
		// Error reading pipelines. Requeue the request.
		return reconcile.Result{}, err
	}

	for _, pipeline := range pipelineList.Items {
		if pipeline.Spec.Trigger.Event.Type != instance.Spec.Type {
			r.log.Info("Ignored with mismatched event type", "pipeline", pipeline.Name)
			r.recorder.Event(instance, "Warning", "InvalidPipeline", fmt.Sprintf("Ignored \"%s/%s\" with mismatched event type", pipeline.Namespace, pipeline.Name))
			continue
		}

		matched, err := regexp.MatchString(pipeline.Spec.Trigger.Event.Source, instance.Spec.Source)
		if err != nil {
			r.log.Error(err, "Ignored with invalid source pattern", "pipeline", pipeline.Name)
			r.recorder.Event(instance, "Warning", "InvalidPipeline", fmt.Sprintf("Ignored \"%s/%s\" with invalid source pattern", pipeline.Namespace, pipeline.Name))
			continue
		}
		if !matched {
			continue
		}

		action := r.newAction(instance, &pipeline)
		actionKey := types.NamespacedName{
			Name:      action.Name,
			Namespace: action.Namespace,
		}

		err = r.Get(context.TODO(), actionKey, action)
		if err != nil {
			if errors.IsNotFound(err) {
				err = r.Create(context.TODO(), action)
				if err != nil {
					return reconcile.Result{}, err
				}

				r.log.Info("Created action", "namespace", action.Namespace, "name", action.Name)
				r.recorder.Event(instance, "Normal", "Created", fmt.Sprintf("Created action %s/%s", action.Namespace, action.Name))
			} else if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	ct := metav1.Now()
	event := instance.DeepCopy()
	event.Status.DispatchTime = &ct

	err = r.Update(context.TODO(), event)
	if err != nil {
		return reconcile.Result{}, err
	}

	r.log.Info("Successfully dispatched", "namespace", event.Namespace, "name", event.Name)
	r.recorder.Event(instance, "Normal", "Dispatched", "Successfully dispatched")

	return reconcile.Result{}, nil
}

func (r *ReconcileEvent) newAction(ev *v1alpha1.Event, pipeline *v1alpha1.Pipeline) *v1alpha1.Action {
	name := v1alpha1.NewID()

	labels := map[string]string{
		v1alpha1.KeyEventName:     ev.Name,
		v1alpha1.KeyPipelineName:  pipeline.Name,
		v1alpha1.KeyTransactionID: name,
	}

	for key, val := range pipeline.ObjectMeta.Labels {
		labels[key] = val
	}

	buildSpec := pipeline.Spec.BuildSpec.DeepCopy()

	action := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pipeline.Namespace,
			Labels:    labels,
		},
		Spec: v1alpha1.ActionSpec{
			BuildSpec: *buildSpec,
			Event: v1alpha1.ActionSpecEvent{
				Name:   ev.Name,
				Type:   ev.Spec.Type,
				Source: ev.Spec.Source,
			},
			Pipeline: v1alpha1.ActionSpecPipeline{
				Name:       pipeline.Name,
				Generation: pipeline.Generation,
			},
			Transaction: v1alpha1.ActionSpecTransaction{
				ID:    name,
				Stage: 1,
			},
		},
	}

	return action
}
