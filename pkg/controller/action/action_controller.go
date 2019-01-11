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
	"io/ioutil"
	"reflect"
	"time"

	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/handler"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/source"

	"github.com/go-logr/logr"
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
	return &ReconcileAction{
		Client: mgr.GetClient(),
		scheme: mgr.GetScheme(),
		api:    kubernetes.NewForConfigOrDie(mgr.GetConfig()),
		log:    logf.Log.WithName("action-controller"),
	}
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
	api    kubernetes.Interface
	log    logr.Logger
}

// Reconcile reads that state of the cluster for a Action object and makes changes based on the state read
// and what is in the Action.Spec
// +kubebuilder:rbac:groups=eventreactor.summerwind.github.io,resources=actions,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=build.knative.dev,resources=builds,verbs=get;list;watch;create;update;patch
// +kubebuilder:rbac:groups=,resources=pods/log,verbs=get;list
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

	if instance.Status.NotificationTime != nil {
		return reconcile.Result{}, nil
	}

	// Start another pipelines
	if instance.IsCompleted() {
		err := r.startPipelines(instance)
		if err != nil {
			return reconcile.Result{}, err
		}

		t := metav1.Now()

		action := instance.DeepCopy()
		action.Status.NotificationTime = &t

		r.log.Info("Updating action", "namespace", action.Namespace, "name", action.Name)
		err = r.Update(context.TODO(), action)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	build := r.newBuild(instance)
	err = controllerutil.SetControllerReference(instance, build, r.scheme)
	if err != nil {
		return reconcile.Result{}, err
	}

	err = r.Get(context.TODO(), request.NamespacedName, build)
	if err != nil {
		if errors.IsNotFound(err) {
			r.log.Info("Creating build", "namespace", build.Namespace, "name", build.Name)
			err = r.Create(context.TODO(), build)
			if err != nil {
				return reconcile.Result{}, err
			}

			return reconcile.Result{}, nil
		} else if err != nil {
			return reconcile.Result{}, err
		}
	}

	// Sync build status
	if !reflect.DeepEqual(instance.Status.BuildStatus, build.Status) {
		action := instance.DeepCopy()
		action.Status.BuildStatus = build.Status

		// Get step logs
		if build.Status.Cluster != nil {
			for i, stepName := range build.Status.StepsCompleted {
				if len(action.Status.StepLogs) > i {
					continue
				}

				stepLog, err := r.getStepLog(build.Status.Cluster.Namespace, build.Status.Cluster.PodName, stepName)
				if err != nil {
					r.log.Error(err, "Unable to get step log", "build", build.Name, "step", stepName)
					continue
				}

				action.Status.StepLogs = append(action.Status.StepLogs, stepLog)
			}
		}

		r.log.Info("Updating action", "namespace", action.Namespace, "name", action.Name)
		err = r.Update(context.TODO(), action)
		if err != nil {
			return reconcile.Result{}, err
		}
	}

	return reconcile.Result{}, nil
}

func (r *ReconcileAction) newBuild(action *v1alpha1.Action) *buildv1alpha1.Build {
	buildSpec := action.Spec.BuildSpec.DeepCopy()

	envVars := []corev1.EnvVar{
		corev1.EnvVar{
			Name:  "ER_EVENT_NAME",
			Value: action.Spec.Event.Name,
		},
		corev1.EnvVar{
			Name:  "ER_EVENT_TYPE",
			Value: action.Spec.Event.Type,
		},
		corev1.EnvVar{
			Name:  "ER_EVENT_SOURCE",
			Value: action.Spec.Event.Source,
		},
		corev1.EnvVar{
			Name:  "ER_PIPELINE_NAME",
			Value: action.Spec.Pipeline.Name,
		},
	}

	if action.Spec.Upstream.Name != "" {
		upstreamEnvVars := []corev1.EnvVar{
			corev1.EnvVar{
				Name:  "ER_UPSTREAM_NAME",
				Value: action.Spec.Upstream.Name,
			},
			corev1.EnvVar{
				Name:  "ER_UPSTREAM_STATUS",
				Value: action.Spec.Upstream.Status,
			},
			corev1.EnvVar{
				Name:  "ER_UPSTREAM_VIA",
				Value: action.Spec.Upstream.Via,
			},
		}
		envVars = append(envVars, upstreamEnvVars...)
	}

	for i, _ := range buildSpec.Steps {
		buildSpec.Steps[i].Env = append(buildSpec.Steps[i].Env, envVars...)
	}
	if buildSpec.Template != nil {
		buildSpec.Template.Env = append(buildSpec.Template.Env, envVars...)
	}

	build := &buildv1alpha1.Build{
		ObjectMeta: metav1.ObjectMeta{
			Name:      action.ObjectMeta.Name,
			Namespace: action.Namespace,
		},
		Spec: *buildSpec,
	}

	return build
}

func (r *ReconcileAction) getStepLog(namespace, podName, containerName string) (string, error) {
	opts := &corev1.PodLogOptions{Container: containerName}
	req := r.api.CoreV1().Pods(namespace).GetLogs(podName, opts)

	readCloser, err := req.Timeout(5 * time.Minute).Stream()
	if err != nil {
		return "", err
	}
	defer readCloser.Close()

	b, err := ioutil.ReadAll(readCloser)
	if err != nil {
		return "", err
	}

	return string(b), nil
}

func (r *ReconcileAction) startPipelines(action *v1alpha1.Action) error {
	opts := &client.ListOptions{Namespace: action.Namespace}

	pipelineList := &v1alpha1.PipelineList{}
	err := r.List(context.TODO(), opts, pipelineList)
	if err != nil {
		return err
	}

	for _, pipeline := range pipelineList.Items {
		// Ignore pipeline of current action to avoid looping
		if pipeline.Name == action.Spec.Pipeline.Name {
			continue
		}

		// Ignore if pipeline trigger is not set
		if pipeline.Spec.Trigger.Pipeline == nil {
			continue
		}

		// Ignore if notification status is not matched
		status := pipeline.Spec.Trigger.Pipeline.Status
		if status != "" && status != action.NotificationStatus() {
			continue
		}

		ls := pipeline.Spec.Trigger.Pipeline.Selector
		selector, err := metav1.LabelSelectorAsSelector(&ls)
		if err != nil {
			return err
		}

		// Ignore if labels does not match the label selector
		if !selector.Matches(labels.Set(action.ObjectMeta.Labels)) {
			continue
		}

		na := r.newAction(action, &pipeline)
		naKey := types.NamespacedName{
			Name:      na.Name,
			Namespace: na.Namespace,
		}

		err = r.Get(context.TODO(), naKey, na)
		if err != nil {
			if errors.IsNotFound(err) {
				r.log.Info("Creating next action", "namespace", na.Namespace, "name", na.Name)
				err = r.Create(context.TODO(), na)
				if err != nil {
					return err
				}
			} else if err != nil {
				return err
			}
		}
	}

	return nil
}

func (r *ReconcileAction) newAction(action *v1alpha1.Action, pipeline *v1alpha1.Pipeline) *v1alpha1.Action {
	name := fmt.Sprintf("%s-%s", action.Spec.Notification.Name, pipeline.Name)

	labels := map[string]string{
		v1alpha1.KeyEventName:    action.Spec.Event.Name,
		v1alpha1.KeyPipelineName: pipeline.Name,
	}

	for key, val := range pipeline.ObjectMeta.Labels {
		labels[key] = val
	}

	via := action.Spec.Upstream.Via
	if via == "" {
		via = action.Spec.Pipeline.Name
	} else {
		via = fmt.Sprintf("%s,%s", via, action.Spec.Pipeline.Name)
	}

	buildSpec := pipeline.Spec.BuildSpec.DeepCopy()

	newAction := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: pipeline.Namespace,
			Labels:    labels,
		},
		Spec: v1alpha1.ActionSpec{
			BuildSpec: *buildSpec,
			Event: v1alpha1.ActionSpecEvent{
				Name:   action.Spec.Event.Name,
				Type:   action.Spec.Event.Type,
				Source: action.Spec.Event.Source,
			},
			Pipeline: v1alpha1.ActionSpecPipeline{
				Name:       pipeline.Name,
				Generation: pipeline.Generation,
			},
			Upstream: v1alpha1.ActionSpecUpstream{
				Name:   action.Name,
				Status: action.NotificationStatus(),
				Via:    via,
			},
			Notification: v1alpha1.ActionSpecNotification{
				Name: v1alpha1.NewName(),
			},
		},
	}

	return newAction
}
