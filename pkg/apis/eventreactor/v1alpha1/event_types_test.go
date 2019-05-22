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

	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageEvent(t *testing.T) {
	now := metav1.Now()

	event := &Event{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "default",
		},
		Spec: EventSpec{
			Type:        "eventreactor.test",
			Source:      "/eventreactor/test/storage-event",
			ID:          "4f6e2a13-592a-4c39-b4e4-b7194f4a4318",
			Time:        &now,
			SchemaURL:   "https://eventreactor.summerwind.github.io",
			ContentType: "application/json",
			Data:        "{\"test\":true}",
		},
	}

	g := gomega.NewGomegaWithT(t)

	// Create new event
	g.Expect(c.Create(context.TODO(), event)).NotTo(gomega.HaveOccurred())

	// Get event
	key := types.NamespacedName{
		Name:      event.Name,
		Namespace: event.Namespace,
	}
	saved := &Event{}
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(event))

	// Update dispatchTime field
	updated := saved.DeepCopy()
	updated.Status.DispatchTime = &now
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	// Get updated event
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(updated))

	// Delete event
	g.Expect(c.Delete(context.TODO(), saved)).NotTo(gomega.HaveOccurred())

	// Confirm event deletion
	g.Expect(c.Get(context.TODO(), key, saved)).To(gomega.HaveOccurred())
}

func TestTypeValidation(t *testing.T) {
	var tests = []struct {
		t     string
		valid bool
	}{
		{"eventreactor.event-type_validation", true},
		{strings.Repeat("n", 1), true},
		{strings.Repeat("n", 63), true},
		{strings.Repeat("n", 64), false},
		{"", false},
		{"event/type/validation", false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		now := metav1.Now()
		ev := &Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("type-validation-%02d", i),
				Namespace: "default",
			},
			Spec: EventSpec{
				Type:        test.t,
				Source:      "/eventreactor/test/type-validation",
				ID:          "4f6e2a13-592a-4c39-b4e4-b7194f4a4318",
				Time:        &now,
				ContentType: "application/json",
				Data:        "{\"test\":true}",
			},
		}

		err := ev.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestSourceValidation(t *testing.T) {
	var tests = []struct {
		source string
		valid  bool
	}{
		{"/eventreactor/test/source-validation", true},
		{"/", true},
		{"", false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		now := metav1.Now()
		ev := &Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("source-validation-%02d", i),
				Namespace: "default",
			},
			Spec: EventSpec{
				Type:        "eventreactor.test",
				Source:      test.source,
				ID:          "4f6e2a13-592a-4c39-b4e4-b7194f4a4318",
				Time:        &now,
				ContentType: "application/json",
				Data:        "{\"test\":true}",
			},
		}

		err := ev.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}

func TestIDValidation(t *testing.T) {
	var tests = []struct {
		id    string
		valid bool
	}{
		{"4f6e2a13-592a-4c39-b4e4-b7194f4a4318", true},
		{"4", true},
		{"", false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		now := metav1.Now()
		ev := &Event{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("id-validation-%02d", i),
				Namespace: "default",
			},
			Spec: EventSpec{
				Type:        "eventreactor.test",
				Source:      "/eventreactor/test/id-validation",
				ID:          test.id,
				Time:        &now,
				ContentType: "application/json",
				Data:        "{\"test\":true}",
			},
		}

		err := ev.Validate()
		if test.valid {
			g.Expect(err).NotTo(gomega.HaveOccurred())
		} else {
			g.Expect(err).To(gomega.HaveOccurred())
		}
	}
}
