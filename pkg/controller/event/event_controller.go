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
	"regexp"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add creates a new Event Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileEvent{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		log:    logf.Log.WithName("event-controller"),
	}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("event-controller", mgr, controller.Options{Reconciler: r})
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
	scheme *runtime.Scheme
	log    logr.Logger
}

// Reconcile reads that state of the cluster for a Event object and makes changes based on the state read
// and what is in the Event.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=events,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=pipelines,verbs=get;list;watch
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=actions,verbs=get;list;watch;create
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

	if instance.Status.Processed {
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
			r.log.Info("Mismatched event label", "pipeline", pipeline.Name)
			continue
		}

		matched, err := regexp.MatchString(pipeline.Spec.Trigger.Event.Source, instance.Spec.Source)
		if err != nil {
			r.log.Error(err, "Invalid source pattern", "pipeline", pipeline.Name)
			continue
		}
		if !matched {
			continue
		}

		action := r.newAction(instance, &pipeline)
		err = controllerutil.SetControllerReference(&pipeline, action, r.scheme)
		if err != nil {
			r.log.Error(err, "Unable to set controller reference")
			return reconcile.Result{}, err
		}

		actionKey := types.NamespacedName{
			Name:      action.Name,
			Namespace: action.Namespace,
		}

		err = r.Get(context.TODO(), actionKey, action)
		if err != nil {
			if errors.IsNotFound(err) {
				r.log.Info("Creating new action", "namespace", action.Namespace, "name", action.Name)
				err = r.Create(context.TODO(), action)
				if err != nil {
					return reconcile.Result{}, err
				}
			} else if err != nil {
				return reconcile.Result{}, err
			}
		}
	}

	ct := metav1.Now()
	event := instance.DeepCopy()
	event.Status.Processed = true
	event.Status.CompletionTime = &ct

	r.log.Info("Updating event", "namespace", event.Namespace, "name", event.Name)
	err = r.Update(context.TODO(), event)
	if err != nil {
		return reconcile.Result{}, err
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileEvent) newAction(ev *v1alpha1.Event, pipeline *v1alpha1.Pipeline) *v1alpha1.Action {
	name := v1alpha1.NewName()

	envVars := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "EVENTREACTOR_EVENT_NAME",
			Value: ev.Name,
		},
		corev1.EnvVar{
			Name:  "EVENTREACTOR_EVENT_TYPE",
			Value: ev.Spec.Type,
		},
		corev1.EnvVar{
			Name:  "EVENTREACTOR_EVENT_SOURCE",
			Value: ev.Spec.Source,
		},
		corev1.EnvVar{
			Name:  "EVENTREACTOR_PIPELINE_NAME",
			Value: pipeline.Name,
		},
		corev1.EnvVar{
			Name:  "EVENTREACTOR_ACTION_NAME",
			Value: name,
		},
	}

	buildSpec := pipeline.Spec.BuildSpec.DeepCopy()

	for i, _ := range buildSpec.Steps {
		buildSpec.Steps[i].Env = append(buildSpec.Steps[i].Env, envVars...)
	}
	if buildSpec.Template != nil {
		buildSpec.Template.Env = append(buildSpec.Template.Env, envVars...)
	}

	labels := map[string]string{
		v1alpha1.KeyEventName:    ev.Name,
		v1alpha1.KeyPipelineName: pipeline.Name,
	}

	for key, val := range pipeline.ObjectMeta.Labels {
		labels[key] = val
	}

	action := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pipeline.Namespace,
			Labels:    labels,
		},
		Spec: v1alpha1.ActionSpec{
			BuildSpec: *buildSpec,
			Event:     ev.Name,
			Pipeline: v1alpha1.ActionSpecPipeline{
				Name:       pipeline.Name,
				Generation: pipeline.Generation,
			},
		},
	}

	return action
}
