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
	"context"
	"errors"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineTriggerEvent defines the condition of the event to execute pipeline.
type PipelineTriggerEvent struct {
	// Type specifies the type of event.
	// +kubebuilder:validation:MinLength=1
	// +kubebuilder:validation:MaxLength=63
	// +kubebuilder:validation:Pattern=^[a-z0-9A-Z\-_.]+$
	Type string `json:"type"`

	// SourcePattern specifies a regular expression pattern
	// that matches the source field of event.
	SourcePattern string `json:"sourcePattern"`
}

// PipelineTriggerPipeline defines the condition of the pipeline to execute
// pipeline.
type PipelineTriggerPipeline struct {
	// Name specifies the name of pipeline.
	Name string `json:"name,omitempty"`

	// Selector is a selector which must be true for the labels of pipeline.
	Selector metav1.LabelSelector `json:"selector,omitempty"`

	// Status specifies the status of pipeline. If the value is empty,
	// it matches both success and failure.
	// +kubebuilder:validation:Enum=success,failure,neutral
	Status CompletionStatus `json:"status,omitempty"`
}

// PipelineTrigger defines the cause of pipeline execution.
type PipelineTrigger struct {
	// Event contains the condition of the event to execute pipeline.
	Event *PipelineTriggerEvent `json:"event,omitempty"`
	// Pipeline contains the condition of the pipeline to execute pipeline.
	Pipeline *PipelineTriggerPipeline `json:"pipeline,omitempty"`
}

// Validate returns error when its field values are invalid.
func (pt PipelineTrigger) Validate() error {
	if pt.Event == nil && pt.Pipeline == nil {
		return errors.New("Trigger must be specified")
	}

	if pt.Event != nil && pt.Pipeline != nil {
		return errors.New("Trigger must be exactly one")
	}

	return nil
}

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	buildv1alpha1.BuildSpec

	// Trigger specifies the trigger of the Pipeline.
	Trigger PipelineTrigger `json:"trigger"`
}

// PipelineStatus defines the observed state of Pipeline.
type PipelineStatus struct{}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Pipeline is the Schema for the pipelines API.
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Pipeline struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   PipelineSpec   `json:"spec,omitempty"`
	Status PipelineStatus `json:"status,omitempty"`
}

// Validate returns error when its field values are invalid.
func (p Pipeline) Validate() error {
	err := p.Spec.Trigger.Validate()
	if err != nil {
		return err
	}

	fieldErr := p.Spec.BuildSpec.Validate(context.Background())
	if fieldErr != nil {
		return fieldErr
	}

	return nil
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// PipelineList contains a list of Pipeline.
type PipelineList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Pipeline `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Pipeline{}, &PipelineList{})
}
