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

type CompletionStatus string

const (
	CompletionStatusSuccess CompletionStatus = "success"
	CompletionStatusFailure CompletionStatus = "failure"
	CompletionStatusNeutral CompletionStatus = "neutral"
	CompletionStatusUnknown CompletionStatus = "unknown"

	ExitCodeNeutral int32 = 78
)

// ActionSpecEvent defines the event information of Action.
// For actions triggered by other actions, this information is copied
// from the original action.
type ActionSpecEvent struct {
	// Name is the name of event.
	Name string `json:"name"`

	// Source is the type of event.
	Type string `json:"type"`

	// Source is the source of event.
	Source string `json:"source"`
}

// ActionSpecPipeline defines the pipeline information of Action.
type ActionSpecPipeline struct {
	// Name is the name of pipeline.
	Name string `json:"name"`

	// Generation is the resource generation of pipeline resource.
	Generation int64 `json:"generation"`
}

// ActionSpecUpstream defines the upstream information of Action.
type ActionSpecUpstream struct {
	// Name specifies the name of upstream action.
	Name string `json:"name"`

	// Status specifies the status of upstream action.
	Status CompletionStatus `json:"status"`

	// Pipeline is the pipeline name of upstream action.
	Pipeline string `json:"pipeline"`

	// Via is a list of upstream action names.
	Via []string `json:"via,omitempty"`
}

// ActionSpecTransaction defines the transaction information of Action.
// If this action triggered another action, this information is also
// carried over to the next action.
type ActionSpecTransaction struct {
	// ID is a unique string used as the identifier of the transaction.
	ID string `json:"id"`

	// Stage specifies the number of actions in the transaction.
	Stage int `json:"stage"`
}

// ActionSpec defines the desired state of Action.
type ActionSpec struct {
	buildv1alpha1.BuildSpec

	// Event contains information of the event.
	Event ActionSpecEvent `json:"event"`

	// Pipeline contains information of the pipeline.
	Pipeline ActionSpecPipeline `json:"pipeline"`

	// Upstream contains information of the action that triggered pipeline.
	Upstream ActionSpecUpstream `json:"upstream"`

	// Transaction is shared information carried over between actions.
	Transaction ActionSpecTransaction `json:"transaction"`
}

// ActionStatus defines the observed state of Action.
type ActionStatus struct {
	buildv1alpha1.BuildStatus

	// StepLogs contains the output log for each step.
	StepLogs []string `json:"stepLogs,omitempty"`

	// DispatchTime specifies the time when this action executed
	// another pipeline by controller.
	DispatchTime *metav1.Time `json:"dispatchTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Action is the Schema for the actions API.
// +k8s:openapi-gen=true
type Action struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ActionSpec   `json:"spec,omitempty"`
	Status ActionStatus `json:"status,omitempty"`
}

// IsCompleted returns true when the action is completed.
func (a Action) IsCompleted() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status != corev1.ConditionUnknown)
}

// IsSucceeded returns true when the action is succeeded.
func (a Action) IsSucceeded() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status == corev1.ConditionTrue)
}

// IsFailed returns true when the action is failed.
func (a Action) IsFailed() bool {
	cond := a.Status.GetCondition(buildv1alpha1.BuildSucceeded)
	return (cond != nil && cond.Status == corev1.ConditionFalse)
}

// CompletionStatus returns "success" or "failure" based on the state of Action.
func (a Action) CompletionStatus() CompletionStatus {
	status := CompletionStatusUnknown

	cond := a.Status.BuildStatus.GetCondition(buildv1alpha1.BuildSucceeded)
	if cond == nil {
		return status
	}

	switch cond.Status {
	case corev1.ConditionTrue:
		status = CompletionStatusSuccess
	case corev1.ConditionFalse:
		status = CompletionStatusFailure
	}

	if status == CompletionStatusFailure && len(a.Status.BuildStatus.StepStates) > 0 {
		var lastExitCode int32

		for _, ss := range a.Status.BuildStatus.StepStates {
			if ss.Terminated != nil {
				lastExitCode = ss.Terminated.ExitCode
			}
		}

		if lastExitCode == ExitCodeNeutral {
			status = CompletionStatusNeutral
		}
	}

	return status
}

// FailedReason returns brief reason of the status
func (a Action) FailedReason() string {
	reason := ""

	cond := a.Status.BuildStatus.GetCondition(buildv1alpha1.BuildSucceeded)
	if cond == nil {
		return reason
	}

	return cond.Reason
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// ActionList contains a list of Action.
type ActionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Action `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Action{}, &ActionList{})
}
