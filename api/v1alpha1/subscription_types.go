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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SubscriptionSpec defines the desired state of Subscription
type SubscriptionSpec struct {
	Trigger SubscriptionSpecTrigger `json:"trigger"`
	// +kubebuilder:validation:MinItems=1
	//ResourceTemplates []runtime.RawExtension `json:"resourceTemplates,omitempty"`
	ResourceTemplates []unstructured.Unstructured `json:"resourceTemplates,omitempty"`
}

// SubscriptionSpecTrigger defines the trigger of Subscription
type SubscriptionSpecTrigger struct {
	Type string `json:"type"`
	// +optional
	MatchSource string `json:"matchSource"`
	// +optional
	MatchSubject string `json:"matchSubject"`
}

// SubscriptionStatus defines the observed state of Subscription
type SubscriptionStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true

// Subscription is the Schema for the subscriptions API
type Subscription struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SubscriptionSpec   `json:"spec,omitempty"`
	Status SubscriptionStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// SubscriptionList contains a list of Subscription
type SubscriptionList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Subscription `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Subscription{}, &SubscriptionList{})
}
