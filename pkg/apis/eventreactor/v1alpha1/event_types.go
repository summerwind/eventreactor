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
	"errors"
	"fmt"
	"regexp"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EventSpec defines the desired state of Event.
type EventSpec struct {
	// Type specifies the type of CloudEvents.
	Type string `json:"type"`

	// Source specifies the source of CloudEvents.
	Source string `json:"source"`

	// ID specifies the unique ID of CloudEvents.
	ID string `json:"id"`

	// Time specifies the time of CloudEvents.
	Time *metav1.Time `json:"time,omitempty"`

	// SchemaURL specifies the schema URL of CloudEvents.
	SchemaURL string `json:"schemaURL,omitempty"`

	// ContentType specifies the type of data.
	ContentType string `json:"contentType,omitempty"`

	// Data specifies the event payload.
	Data string `json:"data,omitempty"`
}

func (e *EventSpec) Validate() error {
	if e.Type == "" {
		return errors.New("type must be specified")
	}

	if len(e.Type) > 63 {
		return errors.New("type is too long")
	}

	matched, err := regexp.MatchString(`^[a-z0-9A-Z\-_.]+$`, e.Type)
	if err != nil {
		return fmt.Errorf("invalid type pattern: %v", err)
	}
	if !matched {
		return errors.New("invalid type")
	}

	if e.Source == "" {
		return errors.New("source must be specified")
	}

	if e.ID == "" {
		return errors.New("id must be specified")
	}

	return nil
}

// EventStatus defines the observed state of Event.
type EventStatus struct {
	// DispatchTime specifies the time of handled the event by ontroller.
	DispatchTime *metav1.Time `json:"dispatchTime,omitempty"`
}

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Event is the Schema for the events API.
// +k8s:openapi-gen=true
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSpec   `json:"spec,omitempty"`
	Status EventStatus `json:"status,omitempty"`
}

// Validate returns error when its field values are invalid.
func (e *Event) Validate() error {
	return e.Spec.Validate()
}

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// EventList contains a list of Event.
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Event `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Event{}, &EventList{})
}
