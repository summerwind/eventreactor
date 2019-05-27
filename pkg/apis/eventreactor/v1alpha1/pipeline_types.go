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
	"fmt"
	"regexp"

	buildv1alpha1 "github.com/knative/build/pkg/apis/build/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// PipelineTriggerEvent defines the condition of the event to execute pipeline.
type PipelineTriggerEvent struct {
	// Type specifies the type of event.
	Type string `json:"type"`

	// SourcePattern specifies a regular expression pattern
	// that matches the source field of event.
	SourcePattern string `json:"sourcePattern"`
}

// Validate returns error when its field values are invalid.
func (p *PipelineTriggerEvent) Validate() error {
	if p.Type == "" {
		return errors.New("type must be specified")
	}

	if len(p.Type) > 63 {
		return errors.New("type is too long")
	}

	matched, err := regexp.MatchString(`^[a-z0-9A-Z\-_.]+$`, p.Type)
	if err != nil {
		return fmt.Errorf("invalid type pattern: %v", err)
	}
	if !matched {
		return errors.New("invalid type")
	}

	_, err = regexp.Compile(p.SourcePattern)
	if err != nil {
		return fmt.Errorf("invalid source pattern: %v", err)
	}

	return nil
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
	Status CompletionStatus `json:"status,omitempty"`
}

// Validate returns error when its field values are invalid.
func (p *PipelineTriggerPipeline) Validate() error {
	switch p.Status {
	case "":
		// Valid
	case CompletionStatusSuccess:
		// Valid
	case CompletionStatusFailure:
		// Valid
	case CompletionStatusNeutral:
		// Valid
	case CompletionStatusUnknown:
		// Valid
	default:
		return fmt.Errorf("invalid pipeline status: %v", p.Status)
	}

	_, err := metav1.LabelSelectorAsSelector(&p.Selector)
	if err != nil {
		return fmt.Errorf("invalid selector: %v", err)
	}

	return nil
}

// PipelineTrigger defines the cause of pipeline execution.
type PipelineTrigger struct {
	// Event contains the condition of the event to execute pipeline.
	Event *PipelineTriggerEvent `json:"event,omitempty"`
	// Pipeline contains the condition of the pipeline to execute pipeline.
	Pipeline *PipelineTriggerPipeline `json:"pipeline,omitempty"`
}

// Validate returns error when its field values are invalid.
func (p *PipelineTrigger) Validate() error {
	if p.Event == nil && p.Pipeline == nil {
		return errors.New("trigger must be specified")
	}

	if p.Event != nil && p.Pipeline != nil {
		return errors.New("trigger must be exactly one")
	}

	if p.Event != nil {
		err := p.Event.Validate()
		if err != nil {
			return err
		}
	}

	if p.Pipeline != nil {
		err := p.Pipeline.Validate()
		if err != nil {
			return err
		}
	}

	return nil
}

// PipelineSpec defines the desired state of Pipeline.
type PipelineSpec struct {
	buildv1alpha1.BuildSpec

	// Trigger specifies the trigger of the Pipeline.
	Trigger PipelineTrigger `json:"trigger"`
}

// Validate returns error when its field values are invalid.
func (p *PipelineSpec) Validate() error {
	err := p.Trigger.Validate()
	if err != nil {
		return err
	}

	fieldErr := p.BuildSpec.Validate(context.Background())
	if fieldErr != nil {
		return fieldErr
	}

	return nil
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
func (p *Pipeline) Validate() error {
	return p.Spec.Validate()
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
