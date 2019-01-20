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

package pipeline

import (
	"context"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
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
)

const (
	ControllerName = "pipeline-controller"
)

// Add creates a new Pipeline Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcilePipeline{
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

	// Watch for changes to Pipeline
	err = c.Watch(&source.Kind{Type: &v1alpha1.Pipeline{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcilePipeline{}

// ReconcilePipeline reconciles a Pipeline object
type ReconcilePipeline struct {
	client.Client
	scheme   *runtime.Scheme
	recorder record.EventRecorder
	log      logr.Logger
}

// Reconcile reads that state of the cluster for a Pipeline object and makes changes based on the state read
// and what is in the Pipeline.Spec
// Automatically generate RBAC rules to allow the Controller to read and write Deployments
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=pipelines,verbs=get;list;watch;update;patch
// +kubebuilder:rbac:groups="",resources=events,verbs=create;patch
func (r *ReconcilePipeline) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	var (
		updated bool
		trigger string
	)

	// Fetch the Pipeline instance
	instance := &v1alpha1.Pipeline{}
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

	pipeline := instance.DeepCopy()

	if pipeline.ObjectMeta.Labels == nil {
		pipeline.ObjectMeta.Labels = map[string]string{}
	}

	if instance.Spec.Trigger.Pipeline != nil {
		trigger = v1alpha1.TriggerTypePipeline
	}

	if instance.Spec.Trigger.Event != nil {
		trigger = v1alpha1.TriggerTypeEvent

		eventType := instance.Spec.Trigger.Event.Type
		val, ok := instance.ObjectMeta.Labels[v1alpha1.KeyEventType]
		if !ok || eventType != val {
			pipeline.ObjectMeta.Labels[v1alpha1.KeyEventType] = eventType
			updated = true
		}
	}

	// No need to update if trigger is empty
	if trigger == "" {
		return reconcile.Result{}, err
	}

	val, ok := instance.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger]
	if !ok || trigger != val {
		pipeline.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = trigger
		updated = true
	}

	if updated {
		err = r.Update(context.TODO(), pipeline)
		if err != nil {
			return reconcile.Result{}, err
		}

		r.log.Info("Updated pipeline", "namespace", pipeline.Namespace, "name", pipeline.Name)
		r.recorder.Event(instance, "Normal", "Labeled", "Successfully labeled")
	}

	return reconcile.Result{}, nil
}
