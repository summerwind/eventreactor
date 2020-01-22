/*
Copyright 2020 The Event Reactor authors.

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
	"math/rand"
	"strings"
	"time"

	"github.com/oklog/ulid/v2"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var entropy *rand.Rand

// EventSpec defines the desired state of Event
type EventSpec struct {
	// ID specifies the unique ID of event.
	ID string `json:"id"`
	// Source specifies the source of event.
	Source string `json:"source"`
	// Type specifies the type of events.
	Type string `json:"type"`

	// DataContentType specifies the content type of data.
	// +optional
	DataContentType string `json:"dataContentType,omitempty"`
	// DataSchema specifies the URL of data schema.
	// +optional
	DataSchema string `json:"dataSchema,omitempty"`
	// Subject specifies the subject of the event in the context of the event producer.
	// +optional
	Subject string `json:"subject,omitempty"`
	// Time specifies the timestamp of when the occurrence happened.
	// +optional
	Time *metav1.Time `json:"time,omitempty"`

	// Data specifies the event payload.
	// +optional
	Data string `json:"data,omitempty"`
}

// EventStatus defines the observed state of Event
type EventStatus struct {
	// The phase of a Event is a simple, high-level summary of where the Event is in its lifecycle.
	Phase string `json:"phase"`
	// A brief CamelCase message indicating details about why the event is in this state.
	Reason string `json:"reason"`
	// A human readable message indicating details about why the event is in this condition.
	Message string `json:"message"`
	// RFC 3339 date and time at which the object was acknowledged by the controller.
	DispatchTime *metav1.Time `json:"dispatchTime,omitempty"`
}

// +kubebuilder:object:root=true

// Event is the Schema for the events API
type Event struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   EventSpec   `json:"spec,omitempty"`
	Status EventStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// EventList contains a list of Event
type EventList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Event `json:"items"`
}

func NewEventName() string {
	id, err := ulid.New(ulid.Now(), entropy)
	if err != nil {
		panic(fmt.Sprintf("Unable to generate ULID: %s", err))
	}

	return strings.ToLower(id.String())
}

func init() {
	SchemeBuilder.Register(&Event{}, &EventList{})

	entropy = rand.New(rand.NewSource(time.Now().UnixNano()))
}
