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
	"testing"

	"github.com/knative/build/pkg/apis/build/v1alpha1"
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	duckv1alpha1 "github.com/knative/pkg/apis/duck/v1alpha1"
	"github.com/onsi/gomega"
	"golang.org/x/net/context"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

func TestStorageAction(t *testing.T) {
	now := metav1.Now()

	action := &Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "01d25bsbhcwhx228s89wmhz37y",
			Namespace: "default",
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
				Source: "/eventreactor/test/storage-action",
			},
			Pipeline: ActionSpecPipeline{
				Name:       "test",
				Generation: 1,
			},
			Transaction: ActionSpecTransaction{
				ID:    "01d2927bszc2yn394z6khwr1p8",
				Stage: 1,
			},
		},
	}

	g := gomega.NewGomegaWithT(t)

	// Create new action
	g.Expect(c.Create(context.TODO(), action)).NotTo(gomega.HaveOccurred())

	// Get action
	key := types.NamespacedName{
		Name:      action.Name,
		Namespace: action.Namespace,
	}
	saved := &Action{}
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(action))

	// Update dispatchTime field
	updated := saved.DeepCopy()
	updated.Status.DispatchTime = &now
	g.Expect(c.Update(context.TODO(), updated)).NotTo(gomega.HaveOccurred())

	// Get updated action
	g.Expect(c.Get(context.TODO(), key, saved)).NotTo(gomega.HaveOccurred())
	g.Expect(saved).To(gomega.Equal(updated))

	// Delete action
	g.Expect(c.Delete(context.TODO(), saved)).NotTo(gomega.HaveOccurred())

	// Confirm action deletion
	g.Expect(c.Get(context.TODO(), key, saved)).To(gomega.HaveOccurred())
}

func TestIsCompleted(t *testing.T) {
	successCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Message: "Success",
	}
	failureCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Message: "Failure",
	}
	unknownCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionUnknown,
		Message: "Unknown",
	}

	var tests = []struct {
		cond  *duckv1alpha1.Condition
		valid bool
	}{
		{successCond, true},
		{failureCond, true},
		{unknownCond, false},
		{nil, false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		action := &Action{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("is-completed-%02d", i),
				Namespace: "default",
			},
			Status: ActionStatus{
				BuildStatus: buildv1alpha1.BuildStatus{},
			},
		}

		action.Status.BuildStatus.SetCondition(test.cond)
		g.Expect(action.IsCompleted()).To(gomega.Equal(test.valid))
	}
}

func TestIsSucceeded(t *testing.T) {
	successCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Message: "Success",
	}
	failureCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Message: "Failure",
	}
	unknownCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionUnknown,
		Message: "Unknown",
	}

	var tests = []struct {
		cond  *duckv1alpha1.Condition
		valid bool
	}{
		{successCond, true},
		{failureCond, false},
		{unknownCond, false},
		{nil, false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		action := &Action{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("is-succeeded-%02d", i),
				Namespace: "default",
			},
			Status: ActionStatus{
				BuildStatus: buildv1alpha1.BuildStatus{},
			},
		}

		action.Status.BuildStatus.SetCondition(test.cond)
		g.Expect(action.IsSucceeded()).To(gomega.Equal(test.valid))
	}
}

func TestIsFailed(t *testing.T) {
	successCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Message: "Success",
	}
	failureCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Message: "Failure",
	}
	unknownCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionUnknown,
		Message: "Unknown",
	}

	var tests = []struct {
		cond  *duckv1alpha1.Condition
		valid bool
	}{
		{successCond, false},
		{failureCond, true},
		{unknownCond, false},
		{nil, false},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		action := &Action{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("is-failed-%02d", i),
				Namespace: "default",
			},
			Status: ActionStatus{
				BuildStatus: buildv1alpha1.BuildStatus{},
			},
		}

		action.Status.BuildStatus.SetCondition(test.cond)
		g.Expect(action.IsFailed()).To(gomega.Equal(test.valid))
	}
}

func TestCompletionStatus(t *testing.T) {
	successCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionTrue,
		Message: "Success",
	}
	failureCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionFalse,
		Message: "Failure",
	}
	unknownCond := &duckv1alpha1.Condition{
		Type:    v1alpha1.BuildSucceeded,
		Status:  corev1.ConditionUnknown,
		Message: "Unknown",
	}

	var tests = []struct {
		cond   *duckv1alpha1.Condition
		status string
	}{
		{successCond, "success"},
		{failureCond, "failure"},
		{unknownCond, ""},
		{nil, ""},
	}

	g := gomega.NewGomegaWithT(t)

	for i, test := range tests {
		action := &Action{
			ObjectMeta: metav1.ObjectMeta{
				Name:      fmt.Sprintf("completion-status-%02d", i),
				Namespace: "default",
			},
			Status: ActionStatus{
				BuildStatus: buildv1alpha1.BuildStatus{},
			},
		}

		action.Status.BuildStatus.SetCondition(test.cond)
		g.Expect(action.CompletionStatus()).To(gomega.Equal(test.status))
	}
}
