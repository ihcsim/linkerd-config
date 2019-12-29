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

package v1alpha1

import (
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// LinkerdConfigSpec defines the desired state of LinkerdConfig
type LinkerdConfigSpec struct {
	// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	Global Global `json:"global"`
	Proxy  Proxy  `json:"proxy"`
}

// LinkerdConfigStatus defines the observed state of LinkerdConfig
type LinkerdConfigStatus struct {
	// INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
	// Important: Run "make" to regenerate code after modifying this file

	// A list of pointers to injected workloads
	// +kubebuilder:validation:Optional
	Injected []corev1.ObjectReference `json:"injected,omitempty"`

	// A list of pointers to uninjected workloads
	// +kubebuilder:validation:Optional
	Uninjected []corev1.ObjectReference `json:"uninjected,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status

// LinkerdConfig is the Schema for the linkerdconfigs API
type LinkerdConfig struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   LinkerdConfigSpec   `json:"spec,omitempty"`
	Status LinkerdConfigStatus `json:"status,omitempty"`
}

// +kubebuilder:object:root=true

// LinkerdConfigList contains a list of LinkerdConfig
type LinkerdConfigList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []LinkerdConfig `json:"items"`
}

// Global contains global configuration of the control plane.
type Global struct {
	LinkerdNamespace       string          `json:"linkerdNamespace"`
	CNIEnabled             bool            `json:"cniEnabled"`
	Version                string          `json:"version"`
	IdentityContext        IdentityContext `json:"identityContext"`
	OmitWebhookSideEffects bool            `json:"omitWebhookSideEffects"`
	ClusterDomain          string          `json:"clusterDomain"`
}

// IdentityContext contains mTLS trust configuration.
type IdentityContext struct {
	TrustDomain        string        `json:"trustDomain"`
	TrustAnchorsPEM    string        `json:"trustAnchorsPEM"`
	IssuanceLifeTime   time.Duration `json:"issuanceLifeTime"`
	ClockSkewAllowance time.Duration `json:"clockSkewAllowance"`
	Scheme             string        `json:"scheme"`
}

// Proxy contains the dataplane's proxy configuration.
type Proxy struct {
	AdminPort               Port      `json:"adminPort"`
	ControlPort             Port      `json:"controlPort"`
	DisableExternalProfiles bool      `json:"disableExternalProfiles"`
	IgnoreInboundPorts      Ports     `json:"ignoreInboundPorts"`
	IgnoreOutboundPorts     Ports     `json:"ignoreOutboundPorts"`
	InboundPort             Port      `json:"inboundPort"`
	LogLevel                string    `json:"logLevel"`
	OutboundPort            Port      `json:"outboundPort"`
	ProxyImage              Image     `json:"proxyImage"`
	ProxyInitImage          Image     `json:"proxyInitImage"`
	ProxyInitImageVersion   string    `json:"proxyInitImageVersion"`
	ProxyVersion            string    `json:"proxyVersion"`
	ProxyUID                uint64    `json:"proxyUID"`
	Resource                Resources `json:"resource"`
}

// Image represents a container's image.
type Image struct {
	Name       string `json:"name"`
	PullPolicy string `json:"pullPolicy"`
}

// Port represents a port number.
type Port int32

var emptyPort = Port(0)

func (p Port) empty() bool {
	return p == emptyPort
}

// Ports is a collection of ports.
type Ports []Port

// Resources represents the resource requests and limits requirements of a container.
type Resources struct {
	// +kubebuilder:validation:Optional
	Limits map[string]string `json:"limits"`

	// +kubebuilder:validation:Optional
	Requests map[string]string `json:"requests"`
}

func init() {
	SchemeBuilder.Register(&LinkerdConfig{}, &LinkerdConfigList{})
}
