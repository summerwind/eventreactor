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
	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const (
	NotificationStatusSuccess = "success"
	NotificationStatusFailure = "failure"
)

type ActionSpecEvent struct {
	Name string `json:"name"`
	Kind string `json:"kind"`
}

type ActionSpecPipeline struct {
	Name       string `json:"name"`
	Generation int64  `json:"generation"`
}

type ActionSpecNotification struct {
	Name string `json:"name"`
}

// ActionSpec defines the desired state of Action
type ActionSpec struct {
	buildv1alpha1.BuildSpec

	Event        ActionSpecEvent        `json:"event"`
	Pipeline     ActionSpecPipeline     `json:"pipeline"`
	Notification ActionSpecNotification `json:"notification"`
}

// ActionStatus defines the observed state of Action
type ActionStatus struct {
	buildv1alpha1.BuildStatus

	StepLogs         []string     `json:"stepLogs,omitempty"`
	NotificationTime *metav1.Time `json:"notificationTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Action is the Schema for the actions API
// +k8s:openapi-gen=true
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec,omitempty"`
	Status ActionStatus `json:"status,omitempty"`
}

func (a Action) IsCompleted() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status != corev1.ConditionUnknown)
}

func (a Action) IsSucceeded() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status == corev1.ConditionTrue)
}

func (a Action) IsFailed() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status == corev1.ConditionFalse)
}

func (a Action) NotificationStatus() string {
	status := ""

	cond := a.Status.BuildStatus.GetCondition(buildv1alpha1.BuildSucceeded)
	if cond == nil {
		return status
	}

	switch cond.Status {
	case corev1.ConditionTrue:
		status = NotificationStatusSuccess
	case corev1.ConditionFalse:
		status = NotificationStatusFailure
	}

	return status
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActionList contains a list of Action
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Action{}, &ActionList{})
}
