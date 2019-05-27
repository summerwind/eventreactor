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

package v1alpha1

import (
	"fmt"
	"strings"
	"testing"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStoragePipeline(t *testing.T) {
	pipeline := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			BuildSpec: buildv1alpha1.BuildSpec{
				Steps: []corev1.Container{
					corev1.Container{
						Name:  "hello",
						Image: "ubuntu:18.04",
						Args:  []string{"echo", "hello world"},
					},
				},
			},

			Trigger: PipelineTrigger{
				Event: &PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: "/eventreactor/test/storage-pipeline",
				},
			},
		},
	}

	g := gomega.NewGomegaWithT(t)

	// Create new pipeline
	g.Expect(c.Create(context.TODO(), pipeline)).NotTo(gomega.HaveOccurred())

	// Get pipeline
	key := types.NamespacedName{
		Name:      pipeline.Name,
		Namespace: pipeline.Namespace,
	}
	saved := &Pipeline{}
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(pipeline))

	// Update labels
	updated := saved.DeepCopy()
	updated.Labels = map[string]string{
		KeyEventType: updated.Spec.Trigger.Event.Type,
	}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	// Get updated pipeline
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(updated))

	// Delete pipeline
	g.Expect(c.Delete(context.TODO(), saved)).NotTo(gomega.HaveOccurred())

	// Confirm pipeline deletion
	g.Expect(c.Get(context.TODO(), key, saved)).To(gomega.HaveOccurred())
}

func TestEventTriggerTypeValidation(t *testing.T) {
	var tests = []struct {
		t     string
		valid bool
	}{
		{"eventreactor.pipleine-type_validation", true},
		{strings.Repeat("n", 1), true},
		{strings.Repeat("n", 63), true},
		{strings.Repeat("n", 64), false},
		{"", false},
		{"pipeline/type/validation", false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		pipeline := &Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("trigger-type-validation-%02d", i),
				Namespace: "default",
			},
			Spec: PipelineSpec{
				BuildSpec: buildv1alpha1.BuildSpec{
					Steps: []corev1.Container{
						corev1.Container{
							Name:  "hello",
							Image: "ubuntu:18.04",
							Args:  []string{"echo", "hello world"},
						},
					},
				},

				Trigger: PipelineTrigger{
					Event: &PipelineTriggerEvent{
						Type:          test.t,
						SourcePattern: "/eventreactor/test/.*",
					},
				},
			},
		}

		err := pipeline.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestPipelineTriggerStatusValidation(t *testing.T) {
	var tests = []struct {
		status CompletionStatus
		valid  bool
	}{
		{CompletionStatusSuccess, true},
		{CompletionStatusFailure, true},
		{CompletionStatusNeutral, true},
		{CompletionStatus(""), true},
		{CompletionStatus("invalid"), false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		pipeline := &Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("trigger-status-validation-%02d", i),
				Namespace: "default",
			},
			Spec: PipelineSpec{
				BuildSpec: buildv1alpha1.BuildSpec{
					Steps: []corev1.Container{
						corev1.Container{
							Name:  "hello",
							Image: "ubuntu:18.04",
							Args:  []string{"echo", "hello world"},
						},
					},
				},

				Trigger: PipelineTrigger{
					Pipeline: &PipelineTriggerPipeline{
						Name:   "test",
						Status: test.status,
					},
				},
			},
		}

		err := pipeline.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestValidate(t *testing.T) {
	valid := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-validate-valid",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{
				Event: &PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: ".+",
				},
			},
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

	invalidNoTrigger := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-validate-invalid-no-trigger",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{},
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

	invalidDoubleTrigger := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-validate-invalid-double-trigger",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{
				Event: &PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: ".+",
				},
				Pipeline: &PipelineTriggerPipeline{
					Name: "test",
				},
			},
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

	invalidBuildSpec := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test-validate-invalid-buildspec",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{
				Event: &PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: ".+",
				},
			},
			BuildSpec: buildv1alpha1.BuildSpec{},
		},
	}

	var tests = []struct {
		pipeline *Pipeline
		valid    bool
	}{
		{valid, true},
		{invalidNoTrigger, false},
		{invalidDoubleTrigger, false},
		{invalidBuildSpec, false},
	}

	g := gomega.NewGomegaWithT(t)

	for _, test := range tests {
		err := test.pipeline.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestNewActionWithEvent(t *testing.T) {
	now := metav1.Now()

	pipeline := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{
				Event: &PipelineTriggerEvent{
					Type:          "eventreactor.test",
					SourcePattern: ".+",
				},
			},
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

	event := &Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: EventSpec{
			Type:        "eventreactor.test",
			Source:      "/eventreactor/test/new-action-with-event",
			ID:          "4f6e2a13-592a-4c39-b4e4-b7194f4a4318",
			Time:        &now,
			SchemaURL:   "https://eventreactor.summerwind.github.io",
			ContentType: "application/json",
			Data:        "{\"test\":true}",
		},
	}

	var tests = []struct {
		trigger *PipelineTriggerEvent
		err     bool
		skip    bool
	}{
		{
			trigger: &PipelineTriggerEvent{
				Type:          "eventreactor.test",
				SourcePattern: ".+",
			},
			err: false,
		},
		{
			trigger: nil,
			err:     true,
		},
		{
			trigger: &PipelineTriggerEvent{
				Type:          "eventreactor.foo",
				SourcePattern: ".+",
			},
			err: true,
		},
		{
			trigger: &PipelineTriggerEvent{
				Type:          "eventreactor.test",
				SourcePattern: "[",
			},
			err: true,
		},
		{
			trigger: &PipelineTriggerEvent{
				Type:          "eventreactor.test",
				SourcePattern: "/eventreactor/foo",
			},
			err: true,
		},
	}

	g := gomega.NewGomegaWithT(t)

	for _, test := range tests {
		pipeline.Spec.Trigger.Event = test.trigger

		a, err := pipeline.NewActionWithEvent(event)
		if test.err {
			g.Expect(err).To(gomega.HaveOccurred())
		} else {
			g.Expect(err).NotTo(gomega.HaveOccurred())

			g.Expect(a.ObjectMeta.Labels[KeyEventName]).To(gomega.Equal(event.Name))
			g.Expect(a.ObjectMeta.Labels[KeyPipelineName]).To(gomega.Equal(pipeline.Name))
			g.Expect(a.ObjectMeta.Labels[KeyTransactionID]).To(gomega.Equal(a.Name))

			g.Expect(a.Namespace).To(gomega.Equal(pipeline.Namespace))
			g.Expect(a.Spec.BuildSpec).To(gomega.Equal(pipeline.Spec.BuildSpec))
			g.Expect(a.Spec.Event.Name).To(gomega.Equal(event.Name))
			g.Expect(a.Spec.Event.Type).To(gomega.Equal(event.Spec.Type))
			g.Expect(a.Spec.Event.Source).To(gomega.Equal(event.Spec.Source))
			g.Expect(a.Spec.Pipeline.Name).To(gomega.Equal(pipeline.Name))
			g.Expect(a.Spec.Pipeline.Generation).To(gomega.Equal(pipeline.Generation))
			g.Expect(a.Spec.Transaction.ID).To(gomega.Equal(a.Name))
			g.Expect(a.Spec.Transaction.Stage).To(gomega.Equal(1))
		}
	}
}

func TestNewActionWithAction(t *testing.T) {
	pipeline := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
			Labels: map[string]string{
				"test": "yes",
			},
		},
		Spec: PipelineSpec{
			Trigger: PipelineTrigger{},
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

	action := &Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "01d25bsbhcwhx228s89wmhz37y",
			Namespace: "default",
			Labels: map[string]string{
				"test": "yes",
			},
		},
		Spec: ActionSpec{
			BuildSpec: buildv1alpha1.BuildSpec{
				Steps: []corev1.Container{
					corev1.Container{
						Name:  "hello",
						Image: "ubuntu:18.04",
						Args:  []string{"echo", "hello world"},
					},
				},
			},
			Event: ActionSpecEvent{
				Name:   "01d29k85cza07ae9taqzbwc1d4",
				Type:   "eventreactor.test",
				Source: "/eventreactor/test/new-action-with-action",
			},
			Pipeline: ActionSpecPipeline{
				Name:       "dummy",
				Generation: 1,
			},
			Transaction: ActionSpecTransaction{
				ID:    "01d2927bszc2yn394z6khwr1p8",
				Stage: 1,
			},
		},
		Status: ActionStatus{
			BuildStatus: buildv1alpha1.BuildStatus{
				StepStates: []corev1.ContainerState{
					corev1.ContainerState{
						Terminated: &corev1.ContainerStateTerminated{
							ExitCode: 0,
						},
					},
				},
			},
		},
	}

	action.Status.BuildStatus.SetCondition(&duckv1alpha1.Condition{
		Type:    buildv1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Message: "Success",
	})

	var tests = []struct {
		trigger  *PipelineTriggerPipeline
		upstream ActionSpecUpstream
		err      bool
	}{
		{
			trigger: &PipelineTriggerPipeline{
				Name:   "dummy",
				Status: CompletionStatusSuccess,
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "yes",
					},
				},
			},
			upstream: ActionSpecUpstream{},
			err:      false,
		},
		{
			trigger:  nil,
			upstream: ActionSpecUpstream{},
			err:      true,
		},
		{
			trigger: &PipelineTriggerPipeline{
				Name: "test",
			},
			upstream: ActionSpecUpstream{},
			err:      true,
		},
		{
			trigger: &PipelineTriggerPipeline{
				Name: "foo",
			},
			upstream: ActionSpecUpstream{},
			err:      true,
		},
		{
			trigger: &PipelineTriggerPipeline{
				Status: CompletionStatusFailure,
			},
			upstream: ActionSpecUpstream{},
			err:      true,
		},
		{
			trigger: &PipelineTriggerPipeline{
				Selector: metav1.LabelSelector{
					MatchLabels: map[string]string{
						"test": "no",
					},
				},
			},
			upstream: ActionSpecUpstream{},
			err:      true,
		},
		{
			trigger: &PipelineTriggerPipeline{
				Name: "dummy",
			},
			upstream: ActionSpecUpstream{
				Name:     "01d2927bszc2yn394z6khwr1p8",
				Status:   "success",
				Pipeline: "test",
				Via:      []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"},
			},
			err: true,
		},
	}

	g := gomega.NewGomegaWithT(t)

	for _, test := range tests {
		pipeline.Spec.Trigger.Pipeline = test.trigger
		action.Spec.Upstream = test.upstream

		a, err := pipeline.NewActionWithAction(action)
		if test.err {
			g.Expect(err).To(gomega.HaveOccurred())
		} else {
			g.Expect(err).NotTo(gomega.HaveOccurred())

			g.Expect(a.ObjectMeta.Labels[KeyEventName]).To(gomega.Equal(action.Spec.Event.Name))
			g.Expect(a.ObjectMeta.Labels[KeyPipelineName]).To(gomega.Equal(pipeline.Name))
			g.Expect(a.ObjectMeta.Labels[KeyTransactionID]).To(gomega.Equal(action.Spec.Transaction.ID))

			g.Expect(a.Namespace).To(gomega.Equal(pipeline.Namespace))
			g.Expect(a.Spec.BuildSpec).To(gomega.Equal(pipeline.Spec.BuildSpec))
			g.Expect(a.Spec.Event.Name).To(gomega.Equal(action.Spec.Event.Name))
			g.Expect(a.Spec.Event.Type).To(gomega.Equal(action.Spec.Event.Type))
			g.Expect(a.Spec.Event.Source).To(gomega.Equal(action.Spec.Event.Source))
			g.Expect(a.Spec.Pipeline.Name).To(gomega.Equal(pipeline.Name))
			g.Expect(a.Spec.Pipeline.Generation).To(gomega.Equal(pipeline.Generation))
			g.Expect(a.Spec.Transaction.ID).To(gomega.Equal(action.Spec.Transaction.ID))
			g.Expect(a.Spec.Transaction.Stage).To(gomega.Equal(action.Spec.Transaction.Stage + 1))
			g.Expect(a.Spec.Upstream.Name).To(gomega.Equal(action.Name))
			g.Expect(a.Spec.Upstream.Status).To(gomega.Equal(action.CompletionStatus()))
			g.Expect(a.Spec.Upstream.Pipeline).To(gomega.Equal(action.Spec.Pipeline.Name))
			g.Expect(a.Spec.Upstream.Via).To(gomega.Equal([]string{action.Spec.Pipeline.Name}))
		}
	}
}
