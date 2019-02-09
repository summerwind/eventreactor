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
	"strings"
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

func newTestPipeline(name string) *v1alpha1.Pipeline {
	return &v1alpha1.Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: "default",
			Labels: map[string]string{
				v1alpha1.KeyPipelineTrigger: v1alpha1.TriggerTypeEvent,
				v1alpha1.KeyEventType:       "eventreactor.test",
				"pipeline":                  "yes",
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
			Trigger: v1alpha1.PipelineTrigger{
				Event: &v1alpha1.PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: "/eventreactor/test",
				},
			},
		},
	}
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

func TestNewBuild(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	r := newReconciler(mgr).(*ReconcileAction)

	action1 := newTestAction("01d25bsbhcwhx228s89wmhz37y")
	action1.Spec.Upstream = v1alpha1.ActionSpecUpstream{
		Name:     "01d25c3gdjedrky8jw529j3wq7",
		Status:   "success",
		Pipeline: "test",
		Via:      []string{"test"},
	}

	build1 := r.newBuild(action1)
	for i, step := range build1.Spec.Steps {
		g.Expect(step.Name).To(gomega.Equal(action1.Spec.BuildSpec.Steps[i].Name))
		g.Expect(step.Image).To(gomega.Equal(action1.Spec.BuildSpec.Steps[i].Image))

		g.Expect(len(step.Env)).To(gomega.Equal(8))
		g.Expect(step.Env[0].Name).To(gomega.Equal("ER_EVENT_NAME"))
		g.Expect(step.Env[0].Value).To(gomega.Equal(action1.Spec.Event.Name))
		g.Expect(step.Env[1].Name).To(gomega.Equal("ER_EVENT_TYPE"))
		g.Expect(step.Env[1].Value).To(gomega.Equal(action1.Spec.Event.Type))
		g.Expect(step.Env[2].Name).To(gomega.Equal("ER_EVENT_SOURCE"))
		g.Expect(step.Env[2].Value).To(gomega.Equal(action1.Spec.Event.Source))
		g.Expect(step.Env[3].Name).To(gomega.Equal("ER_PIPELINE_NAME"))
		g.Expect(step.Env[3].Value).To(gomega.Equal(action1.Spec.Pipeline.Name))
		g.Expect(step.Env[4].Name).To(gomega.Equal("ER_UPSTREAM_NAME"))
		g.Expect(step.Env[4].Value).To(gomega.Equal(action1.Spec.Upstream.Name))
		g.Expect(step.Env[5].Name).To(gomega.Equal("ER_UPSTREAM_STATUS"))
		g.Expect(step.Env[5].Value).To(gomega.Equal(action1.Spec.Upstream.Status))
		g.Expect(step.Env[6].Name).To(gomega.Equal("ER_UPSTREAM_PIPELINE"))
		g.Expect(step.Env[6].Value).To(gomega.Equal(action1.Spec.Upstream.Pipeline))
		g.Expect(step.Env[7].Name).To(gomega.Equal("ER_UPSTREAM_VIA"))
		g.Expect(step.Env[7].Value).To(gomega.Equal(strings.Join(action1.Spec.Upstream.Via, ",")))
	}

	action2 := newTestAction("01d25c3gdjedrky8jw529j3wq7")
	action2.Spec.BuildSpec.Steps = []corev1.Container{}
	action2.Spec.BuildSpec.Template = &buildv1alpha1.TemplateInstantiationSpec{
		Name: "test",
		Kind: buildv1alpha1.BuildTemplateKind,
	}

	build2 := r.newBuild(action2)
	g.Expect(build2.Spec.Template.Name).To(gomega.Equal(action2.Spec.BuildSpec.Template.Name))
	g.Expect(build2.Spec.Template.Kind).To(gomega.Equal(action2.Spec.BuildSpec.Template.Kind))

	env := build2.Spec.Template.Env
	g.Expect(len(env)).To(gomega.Equal(4))
	g.Expect(env[0].Name).To(gomega.Equal("ER_EVENT_NAME"))
	g.Expect(env[0].Value).To(gomega.Equal(action1.Spec.Event.Name))
	g.Expect(env[1].Name).To(gomega.Equal("ER_EVENT_TYPE"))
	g.Expect(env[1].Value).To(gomega.Equal(action1.Spec.Event.Type))
	g.Expect(env[2].Name).To(gomega.Equal("ER_EVENT_SOURCE"))
	g.Expect(env[2].Value).To(gomega.Equal(action1.Spec.Event.Source))
	g.Expect(env[3].Name).To(gomega.Equal("ER_PIPELINE_NAME"))
	g.Expect(env[3].Value).To(gomega.Equal(action1.Spec.Pipeline.Name))
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

func TestStartPipelines(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

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

	a1 := newTestAction("01d25bsbhcwhx228s89wmhz37y")
	a1.Spec.Pipeline.Name = "test1"

	a2 := newTestAction("01d25c3gdjedrky8jw529j3wq7")
	a2.Spec.Pipeline.Name = "test2"
	a2.Spec.Transaction.Stage = 10
	a2.Spec.Upstream = v1alpha1.ActionSpecUpstream{
		Name:     "test",
		Status:   "success",
		Pipeline: "test",
		Via:      []string{"1", "2", "3", "4", "5", "6", "7", "8", "test"},
	}

	// Valid for a1
	p1 := newTestPipeline("valid1")
	p1.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p1.Spec.Trigger.Pipeline = &v1alpha1.PipelineTriggerPipeline{
		Name: a1.Spec.Pipeline.Name,
	}

	// Valid for a2
	p2 := newTestPipeline("valid2")
	p2.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p2.Spec.Trigger.Pipeline = &v1alpha1.PipelineTriggerPipeline{
		Name: a2.Spec.Pipeline.Name,
	}

	// Trigger itself
	p3 := newTestPipeline(a1.Spec.Pipeline.Name)
	p3.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline

	// No trigger
	p4 := newTestPipeline("no-trigger")
	p4.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p4.Spec.Trigger.Event = nil

	// Name does not match
	p5 := newTestPipeline("name-mismatch")
	p5.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p5.Spec.Trigger.Event = nil
	p5.Spec.Trigger.Pipeline = &v1alpha1.PipelineTriggerPipeline{
		Name: "name-mismatch",
	}

	// Status does not match
	p6 := newTestPipeline("status-mismatch")
	p6.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p6.Spec.Trigger.Event = nil
	p6.Spec.Trigger.Pipeline = &v1alpha1.PipelineTriggerPipeline{
		Status: "failure",
	}

	// Status does not match
	p7 := newTestPipeline("selector-mismatch")
	p7.ObjectMeta.Labels[v1alpha1.KeyPipelineTrigger] = v1alpha1.TriggerTypePipeline
	p7.Spec.Trigger.Event = nil
	p7.Spec.Trigger.Pipeline = &v1alpha1.PipelineTriggerPipeline{
		Selector: metav1.LabelSelector{
			MatchLabels: map[string]string{
				"test": "yes",
			},
		},
	}

	// Create pipelines
	g.Expect(c.Create(context.TODO(), p1)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p1)
	g.Expect(c.Create(context.TODO(), p2)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p2)
	g.Expect(c.Create(context.TODO(), p3)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p3)
	g.Expect(c.Create(context.TODO(), p4)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p4)
	g.Expect(c.Create(context.TODO(), p5)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p5)
	g.Expect(c.Create(context.TODO(), p6)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p6)
	g.Expect(c.Create(context.TODO(), p7)).NotTo(gomega.HaveOccurred())
	defer c.Delete(context.TODO(), p7)

	var tests = []struct {
		action   *v1alpha1.Action
		pipeline *v1alpha1.Pipeline
		len      int
	}{
		{a1, p1, 2},
		{a2, nil, 1},
	}

	for _, test := range tests {
		test.action.Status.BuildStatus = newTestBuildStatus(test.action)
		test.action.Status.BuildStatus.SetCondition(&duckv1alpha1.Condition{
			Type:   buildv1alpha1.BuildSucceeded,
			Status: corev1.ConditionTrue,
		})

		req := reconcile.Request{
			NamespacedName: types.NamespacedName{
				Name:      test.action.Name,
				Namespace: test.action.Namespace,
			},
		}

		// Create action
		g.Expect(c.Create(context.TODO(), test.action)).NotTo(gomega.HaveOccurred())
		defer c.Delete(context.TODO(), test.action)

		// Wait for reconcile request by Action creation
		g.Eventually(requests, timeout).Should(gomega.Receive(gomega.Equal(req)))

		// Get actions
		actionList := &v1alpha1.ActionList{}
		opts := &client.ListOptions{Namespace: test.action.Namespace}
		g.Expect(c.List(context.TODO(), opts, actionList)).NotTo(gomega.HaveOccurred())

		fmt.Println(test, actionList.Items)
		g.Expect(len(actionList.Items)).To(gomega.Equal(test.len))
		if test.len > 1 {
			g.Expect(actionList.Items[1].Spec.Pipeline.Name).To(gomega.Equal(test.pipeline.Name))
		}

		for _, a := range actionList.Items {
			g.Expect(c.Delete(context.TODO(), &a)).To(gomega.Succeed())
		}
	}
}

func TestNewAction(t *testing.T) {
	g := gomega.NewGomegaWithT(t)

	mgr, err := manager.New(cfg, manager.Options{})
	g.Expect(err).NotTo(gomega.HaveOccurred())

	r := newReconciler(mgr).(*ReconcileAction)

	action := newTestAction("01d25bsbhcwhx228s89wmhz37y")
	pipeline := newTestPipeline("test-new-action")

	a := r.newAction(action, pipeline)
	labels := a.ObjectMeta.Labels

	g.Expect(labels[v1alpha1.KeyEventName]).To(gomega.Equal(action.Spec.Event.Name))
	g.Expect(labels[v1alpha1.KeyPipelineName]).To(gomega.Equal(pipeline.Name))
	g.Expect(labels[v1alpha1.KeyTransactionID]).To(gomega.Equal(action.Spec.Transaction.ID))
	g.Expect(labels["pipeline"]).To(gomega.Equal("yes"))

	g.Expect(a.Spec.BuildSpec).To(gomega.Equal(pipeline.Spec.BuildSpec))
	g.Expect(a.Spec.Event).To(gomega.Equal(action.Spec.Event))
	g.Expect(a.Spec.Pipeline.Name).To(gomega.Equal(pipeline.Name))
	g.Expect(a.Spec.Pipeline.Generation).To(gomega.Equal(pipeline.Generation))
	g.Expect(a.Spec.Transaction.ID).To(gomega.Equal(action.Spec.Transaction.ID))
	g.Expect(a.Spec.Transaction.Stage).To(gomega.Equal(action.Spec.Transaction.Stage + 1))
	g.Expect(a.Spec.Upstream.Name).To(gomega.Equal(action.Name))
	g.Expect(a.Spec.Upstream.Status).To(gomega.Equal(action.CompletionStatus()))
	g.Expect(a.Spec.Upstream.Pipeline).To(gomega.Equal(action.Spec.Pipeline.Name))
	g.Expect(len(a.Spec.Upstream.Via)).To(gomega.Equal(1))
	g.Expect(a.Spec.Upstream.Via[0]).To(gomega.Equal(action.Spec.Pipeline.Name))
}
