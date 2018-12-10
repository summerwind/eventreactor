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

package action

import (
	"context"
	"fmt"
	"log"
	"reflect"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"sigs.k8s.io/controller-runtime/pkg/source"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	buildscheme "github.com/knative/build/pkg/client/clientset/versioned/scheme"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Add creates a new Action Controller and adds it to the Manager with default RBAC. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &ReconcileAction{Client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Setup Scheme for knative build resources
	buildscheme.AddToScheme(mgr.GetScheme())

	// Create a new controller
	c, err := controller.New("action-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to Action
	err = c.Watch(&source.Kind{Type: &v1alpha1.Action{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// Watch for changes to Build
	err = c.Watch(&source.Kind{Type: &buildv1alpha1.Build{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &v1alpha1.Action{},
	})
	if err != nil {
		fmt.Printf("%v\n", err)
		return err
	}

	return nil
}

var _ reconcile.Reconciler = &ReconcileAction{}

// ReconcileAction reconciles a Action object
type ReconcileAction struct {
	client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a Action object and makes changes based on the state read
// and what is in the Action.Spec
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=actions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.knative.dev,resources=build,verbs=get;list;watch;create;update;patch
func (r *ReconcileAction) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	// Fetch the Action instance
	instance := &v1alpha1.Action{}
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

	cond := instance.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	if cond != nil && cond.Status != corev1.ConditionUnknown {
		return reconcile.Result{}, nil
	}

	build := r.newBuild(instance)
	err = controllerutil.SetControllerReference(instance, build, r.scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.Get(context.TODO(), request.NamespacedName, build)
	if err != nil {
		if errors.IsNotFound(err) {
			log.Printf("Creating Build %s/%s\n", build.Namespace, build.Name)
			err = r.Create(context.TODO(), build)
			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
	}

	if !reflect.DeepEqual(instance.Status.BuildStatus, build.Status) {
		action := instance.DeepCopy()
		action.Status.BuildStatus = build.Status

		log.Printf("Updating Action %s/%s\n", action.Namespace, action.Name)
		err = r.Update(context.TODO(), action)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAction) newBuild(action *v1alpha1.Action) *buildv1alpha1.Build {
	buildSpec := action.Spec.BuildSpec.DeepCopy()

	build := &buildv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      action.ObjectMeta.Name,
			Namespace: action.Namespace,
		},
		Spec: *buildSpec,
	}

	return build
}
