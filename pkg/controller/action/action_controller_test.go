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
	"bytes"
	"fmt"
	"io/ioutil"
	"math/rand"
	"testing"
	"time"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/onsi/gomega"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

const timeout = time.Second * 5

func newTestAction(name string) *v1alpha1.Action {
	return &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
		},
		Spec: v1alpha1.ActionSpec{
			BuildSpec: buildv1alpha1.BuildSpec{
				Steps: []corev1.Container{
					corev1.Container{
						Name:  "hello1",
						Image: "ubuntu:18.04",
						Args:  []string{"echo", "hello world"},
					},
					corev1.Container{
						Name:  "hello2",
						Image: "ubuntu:18.04",
						Args:  []string{"echo", "hello world"},
					},
				},
			},
			Event: v1alpha1.ActionSpecEvent{
				Name:   "01d29k85cza07ae9taqzbwc1d4",
				Type:   "eventreactor.test",
				Source: "/eventreactor/test/action",
			},
			Pipeline: v1alpha1.ActionSpecPipeline{
				Name:       "test",
				Generation: 1,
			},
			Transaction: v1alpha1.ActionSpecTransaction{
				ID:    "01d2927bszc2yn394z6khwr1p8",
				Stage: 1,
			},
		},
	}
}

func newTestBuildStatus(action *v1alpha1.Action) buildv1alpha1.BuildStatus {
	now := metav1.Now()

	status := buildv1alpha1.BuildStatus{
		Builder: buildv1alpha1.ClusterBuildProvider,
		Cluster: &buildv1alpha1.ClusterSpec{
			Namespace: action.Namespace,
			PodName:   action.Name,
		},
		StartTime:      &now,
		CompletionTime: &now,
		StepsCompleted: []string{},
		StepStates:     []corev1.ContainerState{},
	}

	for _, step := range action.Spec.BuildSpec.Steps {
		status.StepsCompleted = append(status.StepsCompleted, fmt.Sprintf("build-step-%s", step.Name))
		status.StepStates = append(status.StepStates, corev1.ContainerState{
			Terminated: &corev1.ContainerStateTerminated{
				ExitCode:    0,
				Reason:      "Completed",
				StartedAt:   now,
				FinishedAt:  now,
				ContainerID: "docker://2e21ff86d823247dff1c35a23198ec5bd69ce782eaedb3277961fc0feb988666",
			},
		})
	}

	return status
}

func TestReconcile(t *testing.T) {
	instance := newTestAction("01d25c3gdjedrky8jw529j3wq7")

	expected := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
	}

	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller. Wrap the Controller Reconcile function
	// so it writes each request to a channel when it is finished.
	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())
	c = mgr.GetClient()

	recFn, requests := SetupTestReconcile(newReconciler(mgr))
	g.Expect(add(mgr, recFn)).NotTo(gomega.HaveOccurred())

	stopMgr, mgrStopped := StartTestManager(mgr, g)

	defer func() {
		close(stopMgr)
		mgrStopped.Wait()
	}()

	// Create action
	g.Expect(c.Create(context.TODO(), instance)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)

	// Wait for reconcile request by Action creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))
	// Wait for reconcile request by Build creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	// Get build
	build := &buildv1alpha1.Build{}
	g.Expect(c.Get(context.TODO(), expected.NamespacedName, build)).To(gomega.Succeed())

	// Test the build spec
	g.Expect(build.Spec.Steps[0].Name).To(gomega.Equal(instance.Spec.Steps[0].Name))
	g.Expect(build.Spec.Steps[0].Image).To(gomega.Equal(instance.Spec.Steps[0].Image))

	// Update build status
	b := build.DeepCopy()
	b.Status = newTestBuildStatus(instance)
	b.Status.SetCondition(&duckv1alpha1.Condition{
		Type:   buildv1alpha1.BuildSucceeded,
		Status: corev1.ConditionTrue,
	})
	g.Expect(c.Status().Update(context.TODO(), b)).To(gomega.Succeed())

	// Wait for reconcile request by Build creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))
	// Wait for reconcile request by Action creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	// Get build
	action := &v1alpha1.Action{}
	g.Expect(c.Get(context.TODO(), expected.NamespacedName, action)).To(gomega.Succeed())

	// Test the action status
	g.Expect(action.Status.DispatchTime).NotTo(gomega.Equal(nil))

	// Manually delete build since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), build)).To(gomega.Succeed())
}

func TestGetStepLog(t *testing.T) {
	const letters = "abcdefghijklmnopqrstuvwxyz"

	var tests = []struct {
		byteLen int
		logLen  int
	}{
		{65536, 65536},
		{65537, 65536},
	}

	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	r := newReconciler(mgr).(*ReconcileAction)

	for _, test := range tests {
		// Generate random bytes
		b := make([]byte, test.byteLen)
		for i := range b {
			b[i] = letters[rand.Intn(len(letters))]
		}
		str := string(b)

		// Set internal log reader
		logReader = ioutil.NopCloser(bytes.NewReader(b))

		// Get step log
		log, err := r.getStepLog("default", "test", "hello")

		g.Expect(err).NotTo(gomega.HaveOccurred())
		g.Expect(len(log)).To(gomega.Equal(test.logLen))
		g.Expect(log[len(log)-10:]).To(gomega.Equal(str[len(str)-10:]))
	}
}
