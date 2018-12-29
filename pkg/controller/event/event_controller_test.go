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
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega"
	v1alpha1 "github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"golang.org/x/net/context"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
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
			Type:   "io.github.summerwind.eventreactor.test",
			Source: "/eventreactor/test/*",
			ID:     "f378179e-7d49-4078-84ce-e529de6dfdca",
			Time:   &now,
		},
	}

	pipeline := &v1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "pipeline",
			Namespace: "default",
			Labels: map[string]string{
				v1alpha1.LabelEventType: "io.github.summerwind.eventreactor.test",
			},
		},
		Spec: v1alpha1.PipelineSpec{
			Trigger: v1alpha1.PipelineTrigger{
				Event: v1alpha1.PipelineEventTrigger{
					Type:   "io.github.summerwind.eventreactor.test",
					Source: "/eventreactor/test/*",
				},
			},
		},
	}

	actionKey := types.NamespacedName{
		Name:      fmt.Sprintf("%s-%s", instance.Name, pipeline.Name),
		Namespace: "default",
	}

	expected := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
	}

	g := gomega.NewGomegaWithT(t)

	// Setup the Manager and Controller.  Wrap the Controller Reconcile function so it writes each request to a
	// channel when it is finished.
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

	err = c.Create(context.TODO(), pipeline)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create pipeline, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), pipeline)

	err = c.Create(context.TODO(), instance)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)

	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	action := &v1alpha1.Action{}
	g.Expect(c.Get(context.TODO(), actionKey, action)).To(gomega.Succeed())

	// Manually delete Deployment since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), action)).To(gomega.Succeed())
}
