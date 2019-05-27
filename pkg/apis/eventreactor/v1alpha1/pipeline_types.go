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
	"k8s.io/apimachinery/pkg/labels"
)

const (
	MaxUpstreamLimit = 9
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

// NewActionWithEvent returns Action based on specified Event.
func (p *Pipeline) NewActionWithEvent(event *Event) (*Action, error) {
	if p.Spec.Trigger.Event == nil {
		return nil, errors.New("event trigger is not set")
	}

	if p.Spec.Trigger.Event.Type != event.Spec.Type {
		return nil, errors.New("event type mismatched")
	}

	matched, err := regexp.MatchString(p.Spec.Trigger.Event.SourcePattern, event.Spec.Source)
	if err != nil {
		return nil, errors.New("invalid source pattern")
	}
	if !matched {
		return nil, errors.New("source pattern mismatched")
	}

	name := NewID()

	labels := map[string]string{}
	for key, val := range p.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[KeyEventName] = event.Name
	labels[KeyPipelineName] = p.Name
	labels[KeyTransactionID] = name

	return &Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: p.Namespace,
			Labels:    labels,
		},
		Spec: ActionSpec{
			BuildSpec: *(p.Spec.BuildSpec.DeepCopy()),
			Event: ActionSpecEvent{
				Name:   event.Name,
				Type:   event.Spec.Type,
				Source: event.Spec.Source,
			},
			Pipeline: ActionSpecPipeline{
				Name:       p.Name,
				Generation: p.Generation,
			},
			Transaction: ActionSpecTransaction{
				ID:    name,
				Stage: 1,
			},
		},
	}, nil
}

// NewActionWithAction returns Action based on specified Action.
func (p *Pipeline) NewActionWithAction(action *Action) (*Action, error) {
	if p.Spec.Trigger.Pipeline == nil {
		return nil, errors.New("pipeline trigger is not set")
	}

	// Ignore if the pipeline is the same as the triggered action
	// to avoid looping
	if p.Name == action.Spec.Pipeline.Name {
		return nil, errors.New("run of same pipeline is not allowed")
	}

	// Ignore if name is not matched
	pn := p.Spec.Trigger.Pipeline.Name
	if pn != "" && pn != action.Spec.Pipeline.Name {
		return nil, errors.New("pipeline name mismatched")
	}

	// Ignore if status is not matched
	status := p.Spec.Trigger.Pipeline.Status
	if status != "" && status != action.CompletionStatus() {
		return nil, errors.New("status mismatched")
	}

	ls := p.Spec.Trigger.Pipeline.Selector
	selector, err := metav1.LabelSelectorAsSelector(&ls)
	if err != nil {
		return nil, err
	}

	// Ignore if labels does not match the selector
	if !selector.Empty() && !selector.Matches(labels.Set(action.ObjectMeta.Labels)) {
		return nil, errors.New("selector mismatched")
	}

	via := action.Spec.Upstream.Via
	if via == nil {
		via = []string{}
	}
	via = append(via, action.Spec.Pipeline.Name)

	if len(via) >= MaxUpstreamLimit {
		return nil, errors.New("upstream limit exceeded")
	}

	labels := map[string]string{}
	for key, val := range p.ObjectMeta.Labels {
		labels[key] = val
	}
	labels[KeyEventName] = action.Spec.Event.Name
	labels[KeyPipelineName] = p.Name
	labels[KeyTransactionID] = action.Spec.Transaction.ID

	return &Action{
		ObjectMeta: metav1.ObjectMeta{
			Name:      NewID(),
			Namespace: p.Namespace,
			Labels:    labels,
		},
		Spec: ActionSpec{
			BuildSpec: *(p.Spec.BuildSpec.DeepCopy()),
			Event:     *(action.Spec.Event.DeepCopy()),
			Pipeline: ActionSpecPipeline{
				Name:       p.Name,
				Generation: p.Generation,
			},
			Transaction: ActionSpecTransaction{
				ID:    action.Spec.Transaction.ID,
				Stage: action.Spec.Transaction.Stage + 1,
			},
			Upstream: ActionSpecUpstream{
				Name:     action.Name,
				Status:   action.CompletionStatus(),
				Pipeline: action.Spec.Pipeline.Name,
				Via:      via,
			},
		},
	}, nil
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
