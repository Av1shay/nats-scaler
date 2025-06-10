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

// ScalingRuleSpec defines the desired state of ScalingRule.
type ScalingRuleSpec struct {
	// +kubebuilder:validation:MinLength=1
	DeploymentName string `json:"deploymentName"`

	// +kubebuilder:validation:MinLength=1
	Namespace string `json:"namespace"`

	// +kubebuilder:validation:Minimum=0
	MinReplicas int32 `json:"minReplicas"`

	// +kubebuilder:validation:Minimum=1
	MaxReplicas int32 `json:"maxReplicas"`

	// +kubebuilder:validation:Pattern=`^https?://`
	NatsMonitoringURL string `json:"natsMonitoringURL"`

	// +kubebuilder:validation:MinLength=1
	StreamName string `json:"streamName"`

	// +kubebuilder:validation:MinLength=1
	ConsumerName string `json:"consumerName"`

	// +kubebuilder:validation:Minimum=0
	ScaleUpThreshold int `json:"scaleUpThreshold"`

	// +kubebuilder:validation:Minimum=0
	ScaleDownThreshold int `json:"scaleDownThreshold"`

	// +kubebuilder:validation:Minimum=1
	PollIntervalSeconds int `json:"pollIntervalSeconds"`
}

// ScalingRuleStatus defines the observed state of ScalingRule.
type ScalingRuleStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file
}

// +kubebuilder:object:root=true
// +kubebuilder:subresource:status

// ScalingRule is the Schema for the scalingrules API.
type ScalingRule struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   ScalingRuleSpec   `json:"spec,omitempty"`
	Status ScalingRuleStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// ScalingRuleList contains a list of ScalingRule.
type ScalingRuleList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []ScalingRule `json:"items"`
}

func init() {
	SchemeBuilder.Register(&ScalingRule{}, &ScalingRuleList{})
}
