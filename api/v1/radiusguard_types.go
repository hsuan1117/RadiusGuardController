/*
Copyright 2025.

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

package v1

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// RADIUSClient Defines the RADIUS Client Object
type RADIUSClient struct {
	// IP address or CIDR of the client
	IPAddress string `json:"ipAddress"`

	// Shared secret for this client
	Secret string `json:"secret"`

	// Friendly name for the client
	Name string `json:"name,omitempty"`
}

// RADIUSGuardSpec defines the desired state of RADIUSGuard
type RADIUSGuardSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	// The following markers will use OpenAPI v3 schema to validate the value
	// More info: https://book.kubebuilder.io/reference/markers/crd-validation.html

	Clients []RADIUSClient `json:"clients"`

	// +optional
	BackendServiceName string `json:"backendServiceName,omitempty"`

	// Custom shared secret for communicating with backend replicas
	BackendSecret string `json:"backendSecret,omitempty"`

	// +optional
	// +kubebuilder:default=1812
	AuthPort int `json:"authPort,omitempty"`

	// +optional
	// +kubebuilder:default=1813
	AcctPort int `json:"acctPort,omitempty"`

	// +optional
	BackendSelector map[string]string `json:"backendSelector,omitempty"`
}

// RADIUSGuardStatus defines the observed state of RADIUSGuard.
type RADIUSGuardStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// For Kubernetes API conventions, see:
	// https://github.com/kubernetes/community/blob/master/contributors/devel/sig-architecture/api-conventions.md#typical-status-properties

	// conditions represent the current state of the RADIUSGuard resource.
	// Each condition has a unique type and reflects the status of a specific aspect of the resource.
	//
	// Standard condition types include:
	// - "Available": the resource is fully functional
	// - "Progressing": the resource is being created or updated
	// - "Degraded": the resource failed to reach or maintain its desired state
	//
	// The status of each condition is one of True, False, or Unknown.
	// +listType=map
	// +listMapKey=type
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// RADIUSGuard is the Schema for the radiusguards API
type RADIUSGuard struct {
	metav1.TypeMeta `json:",inline"`

	// metadata is a standard object metadata
	// +optional
	metav1.ObjectMeta `json:"metadata,omitempty,omitzero"`

	// spec defines the desired state of RADIUSGuard
	// +required
	Spec RADIUSGuardSpec `json:"spec"`

	// status defines the observed state of RADIUSGuard
	// +optional
	Status RADIUSGuardStatus `json:"status,omitempty,omitzero"`
}

// +kubebuilder:object:root=true

// RADIUSGuardList contains a list of RADIUSGuard
type RADIUSGuardList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []RADIUSGuard `json:"items"`
}

func init() {
	SchemeBuilder.Register(&RADIUSGuard{}, &RADIUSGuardList{})
}
