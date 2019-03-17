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
	"testing"
	"time"

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	v1alpha1 "github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var c client.Client

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	now := metav1.Now()

	instance := &v1alpha1.Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.EventSpec{
			Type:   "eventreactor.test",
			Source: "/eventreactor/test/reconcile",
			ID:     "f378179e-7d49-4078-84ce-e529de6dfdca",
			Time:   &now,
		},
	}

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

	var pipelines = []struct {
		name        string
		valid       bool
		eventType   string
		eventSource string
	}{
		{"test-valid", true, "eventreactor.test", "/eventreactor/test/.*"},
		{"test-invalid", false, "eventreactor.test", "/eventreactor/test/.*"},
		{"test-type-mismatched", true, "eventreactor.dummy", "/eventreactor/test/.*"},
		{"test-source-mismatched", true, "eventreactor.test", "/eventreactor/dummy/.*"},
	}

	for _, p := range pipelines {
		pipeline := &v1alpha1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      p.name,
				Namespace: "default",
				Labels: map[string]string{
					v1alpha1.KeyEventType: "eventreactor.test",
					"test":                "yes",
				},
			},
			Spec: v1alpha1.PipelineSpec{
				BuildSpec: buildv1alpha1.BuildSpec{
					Steps: []corev1.Container{
						corev1.Container{
							Name:  "hello",
							Image: "ubuntu:18.04",
							Args:  []string{"echo", "hello world"},
						},
					},
				},
			},
		}

		if p.valid {
			pipeline.Spec.Trigger = v1alpha1.PipelineTrigger{
				Event: &v1alpha1.PipelineTriggerEvent{
					Type:          p.eventType,
					SourcePattern: p.eventSource,
				},
			}
		}

		g.Expect(c.Create(context.TODO(), pipeline)).NotTo(gomega.HaveOccurred())
		defer c.Delete(context.TODO(), pipeline)
	}

	// Craete event
	g.Expect(c.Create(context.TODO(), instance)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)

	// Wait for reconcile request
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	// Get actions
	actionList := &v1alpha1.ActionList{}
	labels := map[string]string{
		v1alpha1.KeyEventName: instance.Name,
	}
	opts := &client.ListOptions{Namespace: instance.Namespace}
	opts = opts.MatchingLabels(labels)

	g.Expect(c.List(context.TODO(), opts, actionList)).To(gomega.Succeed())
	g.Expect(len(actionList.Items)).To(gomega.Equal(1))

	// Test the value of action
	action := actionList.Items[0]
	g.Expect(action.ObjectMeta.Labels[v1alpha1.KeyEventName]).To(gomega.Equal(instance.Name))
	g.Expect(action.ObjectMeta.Labels[v1alpha1.KeyPipelineName]).To(gomega.Equal(pipelines[0].name))
	g.Expect(action.ObjectMeta.Labels[v1alpha1.KeyTransactionID]).NotTo(gomega.Equal(""))
	g.Expect(action.ObjectMeta.Labels["test"]).To(gomega.Equal("yes"))
	g.Expect(action.Spec.Event.Name).To(gomega.Equal(instance.Name))
	g.Expect(action.Spec.Event.Type).To(gomega.Equal(instance.Spec.Type))
	g.Expect(action.Spec.Event.Source).To(gomega.Equal(instance.Spec.Source))
	g.Expect(action.Spec.Pipeline.Name).To(gomega.Equal(pipelines[0].name))
	g.Expect(action.Spec.Transaction.ID).NotTo(gomega.Equal(""))
	g.Expect(action.Spec.Transaction.Stage).To(gomega.Equal(1))

	// Manually delete action since GC isn't enabled in the test control plane
	for _, action := range actionList.Items {
		g.Expect(c.Delete(context.TODO(), &action)).To(gomega.Succeed())
	}
}
