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

		err := c.Create(context.TODO(), pipeline)
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestPipelineTriggerStatusValidation(t *testing.T) {
	var tests = []struct {
		status string
		valid  bool
	}{
		{"success", true},
		{"failure", true},
		{"", true},
		{"invalid", false},
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

		err := c.Create(context.TODO(), pipeline)
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}
