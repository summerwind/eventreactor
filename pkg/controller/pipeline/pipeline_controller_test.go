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
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	v1alpha1 "github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"golang.org/x/net/context"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

var c client.Client

const timeout = time.Second * 5

func TestReconcile(t *testing.T) {
	eventTrigger := v1alpha1.PipelineTrigger{
		Event: &v1alpha1.PipelineTriggerEvent{
			Type:          "eventreactor.test",
			SourcePattern: "/eventreactor/test/reconcile",
		},
	}

	pipelineTrigger := v1alpha1.PipelineTrigger{
		Pipeline: &v1alpha1.PipelineTriggerPipeline{
			Name: "test-trigger",
		},
	}

	emptyTrigger := v1alpha1.PipelineTrigger{}

	var tests = []struct {
		trigger          v1alpha1.PipelineTrigger
		labelTriggerType string
		labelEventType   string
	}{
		{eventTrigger, v1alpha1.TriggerTypeEvent, eventTrigger.Event.Type},
		{pipelineTrigger, v1alpha1.TriggerTypePipeline, ""},
		{emptyTrigger, "", ""},
	}

	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function
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

	for i, test := range tests {
		instance := &v1alpha1.Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("test-%02d", i),
				Namespace: "default",
			},
			Spec: v1alpha1.PipelineSpec{
				Trigger: test.trigger,
			},
		}

		expected := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      instance.Name,
				Namespace: instance.Namespace,
			},
		}

		// Create the Pipeline object and expect the Reconcile and Deployment to be created.
		err = c.Create(context.TODO(), instance)
		if errors.IsInvalid(err) {
			t.Logf("failed to create object, got an invalid object error: %v", err)
			return
		}
		g.Expect(err).NotTo(gomega.HaveOccurred())

		// Wait for reconcile request
		g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

		// Get pipeline
		pipeline := &v1alpha1.Pipeline{}
		g.Expect(c.Get(context.TODO(), expected.NamespacedName, pipeline)).To(gomega.Succeed())

		// Test the label value of pipeline
		labels := pipeline.ObjectMeta.Labels
		g.Expect(labels[v1alpha1.KeyPipelineTrigger]).To(gomega.Equal(test.labelTriggerType))
		g.Expect(labels[v1alpha1.KeyEventType]).To(gomega.Equal(test.labelEventType))

		// Delete pipeline
		g.Expect(c.Delete(context.TODO(), instance)).NotTo(gomega.HaveOccurred())
	}
}
