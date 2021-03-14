/*


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

// CRUDSpec defines the desired state of CRUD
type CRUDSpec struct {
	// +kubebuilder:validation:Required
	APIDescription string `json:"apiDescription"`
	// +kubebuilder:validation:Required
	DomainPrefix string `json:"domainPrefix"`
	// +kubebuilder:default:=true
	EnableTLS bool `json:"enableTLS"`
}

// CRUDStatus defines the observed state of CRUD
type CRUDStatus struct {
	// +kubebuilder:default:=false
	ImageReady         bool   `json:"imageReady"`
	Image              string `json:"image,omitempty"`
	Port               int32  `json:"port,omitempty"`
	APIDescriptionHash string `json:"apiDescriptionHash,omitempty"`
	Deployed           bool   `json:"deployed"`
}

// +kubebuilder:object:root=true
// +kubebuilder:printcolumn:name="Image",type="string",JSONPath=".status.image"
// +kubebuilder:printcolumn:name="Ready",type="boolean",JSONPath=".status.imageReady"
// +kubebuilder:printcolumn:name="Deployed",type="boolean",JSONPath=".status.deployed"
// +kubebuilder:printcolumn:name="Age",type="date",JSONPath=".metadata.creationTimestamp"

// CRUD is the Schema for the cruds API
type CRUD struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   CRUDSpec   `json:"spec,omitempty"`
	Status CRUDStatus `json:"status,omitempty"`
}

func (c *CRUD) LabelSelectors() map[string]string {
	return map[string]string{
		"api.crudgen.org/selector": c.Name,
	}
}

// +kubebuilder:object:root=true

// CRUDList contains a list of CRUD
type CRUDList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []CRUD `json:"items"`
}

func init() {
	SchemeBuilder.Register(&CRUD{}, &CRUDList{})
}
