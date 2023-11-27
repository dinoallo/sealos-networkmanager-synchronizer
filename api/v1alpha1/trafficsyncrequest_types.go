/*
Copyright 2023.

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
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// TrafficSyncRequestSpec defines the desired state of TrafficSyncRequest
type TrafficSyncRequestSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// Foo is an example field of TrafficSyncRequest. Edit trafficsyncrequest_types.go to remove/update
	AssociatedNamespace string          `json:"associatedNamespace,omitempty"`
	AssociatedPod       string          `json:"associatedPod,omitempty"`
	CiliumEndpointID    int64           `json:"ciliumEndpointID,omitempty"`
	NodeIP              string          `json:"nodeIP,omitempty"`
	Address             string          `json:"address,omitempty"`
	Tags                []string        `json:"tags,omitempty"`
	SyncPeriod          metav1.Duration `json:"syncPeriod,omitempty"`
}

// TrafficSyncRequestStatus defines the observed state of TrafficSyncRequest
type TrafficSyncRequestStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
	LastSyncTime map[string]metav1.Time `json:"lastSyncTime,omitempty"`
}

//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// TrafficSyncRequest is the Schema for the trafficsyncrequests API
type TrafficSyncRequest struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   TrafficSyncRequestSpec   `json:"spec,omitempty"`
	Status TrafficSyncRequestStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// TrafficSyncRequestList contains a list of TrafficSyncRequest
type TrafficSyncRequestList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []TrafficSyncRequest `json:"items"`
}

func init() {
	SchemeBuilder.Register(&TrafficSyncRequest{}, &TrafficSyncRequestList{})
}
