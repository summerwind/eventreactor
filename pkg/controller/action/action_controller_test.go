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
	"testing"
	"time"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	"github.com/onsi/gomega"
	"github.com/summerwind/eventreactor/pkg/apis/eventreactor/v1alpha1"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
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
	instance := &v1alpha1.Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: v1alpha1.ActionSpec{
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

	expected := reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      "test",
			Namespace: "default",
		},
	}

	buildKey := types.NamespacedName{
		Name:      "test",
		Namespace: "default",
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

	err = c.Create(context.TODO(), instance)
	if apierrors.IsInvalid(err) {
		t.Logf("failed to create object, got an invalid object error: %v", err)
		return
	}
	g.Expect(err).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), instance)

	// Expect to be called for Action creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))
	// Expect to be called for Build creation
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))

	build := &buildv1alpha1.Build{}
	g.Expect(c.Get(context.TODO(), buildKey, build)).To(gomega.Succeed())

	// Delete the Build and expect Reconcile to be called for Build deletion
	g.Expect(c.Delete(context.TODO(), build)).NotTo(gomega.HaveOccurred())
	g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(expected)))
	g.Expect(c.Get(context.TODO(), buildKey, build)).To(gomega.Succeed())

	// Manually delete Build since GC isn't enabled in the test control plane
	g.Expect(c.Delete(context.TODO(), build)).To(gomega.Succeed())
}
