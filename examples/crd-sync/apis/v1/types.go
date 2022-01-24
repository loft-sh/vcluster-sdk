package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// +genclient
// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// Car holds the webhook configuration
// +k8s:openapi-gen=true
// +kubebuilder:subresource:status
type Car struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CarSpec   `json:"spec,omitempty"`
	Status CarStatus `json:"status,omitempty"`
}

type CarSpec struct {
	// Type is the type of the car
	Type CarType `json:"type,omitempty"`

	// Number of seats within the car
	// +optional
	Seats int `json:"seats,omitempty"`
}

type CarStatus struct {
}

// CarType is the type of the car
type CarType string

const (
	BMW   CarType = "BMW"
	Tesla CarType = "Tesla"
	Audi  CarType = "Audi"
)

// +k8s:deepcopy-gen:interfaces=k8s.io/apimachinery/pkg/runtime.Object

// CarList contains a list of cars
type CarList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Car `json:"items"`
}

func init() {
	SchemeBuilder.Register(&Car{}, &CarList{})
}
