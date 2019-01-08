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
	key := types.NamespacedName{
		Name:      "foo",
		Namespace: "default",
	}
	created := &Pipeline{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "foo",
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
				Event: PipelineTriggerEvent{
					Type:   "io.github.summerwind.eventreactor.test",
					Source: "/eventreactor/test/hello",
				},
			},
		},
	}
	g := gomega.NewGomegaWithT(t)

	// Test Create
	fetched := &Pipeline{}
	g.Expect(c.Create(context.TODO(), created)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(created))

	// Test Updating the Labels
	updated := fetched.DeepCopy()
	updated.Labels = map[string]string{"hello": "world"}
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	g.Expect(c.Get(context.TODO(), key, fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(fetched).To(gomega.Equal(updated))

	// Test Delete
	g.Expect(c.Delete(context.TODO(), fetched)).NotTo(gomega.HaveOccurred())
	g.Expect(c.Get(context.TODO(), key, fetched)).To(gomega.HaveOccurred())
}

func TestEventTriggerTypeValidation(t *testing.T) {
	var tests = []struct {
		t     string
		valid bool
	}{
		{"foo-bar_baz.", true},
		{strings.Repeat("n", 1), true},
		{strings.Repeat("n", 63), true},
		{"", false},
		{strings.Repeat("n", 64), false},
		{"foo/bar/baz", false},
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
					Event: PipelineTriggerEvent{
						Type:   test.t,
						Source: "/eventreactor/test/hello",
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

func TestEventTriggerSourceValidation(t *testing.T) {
	var tests = []struct {
		source string
		valid  bool
	}{
		{"/eventreactor/test/hello", true},
		{"/", true},
		{"", false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		pipeline := &Pipeline{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("trigger-source-validation-%02d", i),
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
					Event: PipelineTriggerEvent{
						Type:   "io.github.summerwind.eventreactor.test",
						Source: test.source,
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
