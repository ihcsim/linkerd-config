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

	"github.com/gogo/protobuf/jsonpb"
	"github.com/golang/protobuf/ptypes"
	"github.com/ihcsim/linkerd-config/proto"
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

	// +kubebuilder:validation:Optional
	// A list of pointers to injected workloads
	Injected []corev1.ObjectReference `json:"injected,omitempty"`

	// +kubebuilder:validation:Optional
	// A list of pointers to uninjected workloads
	Uninjected []corev1.ObjectReference `json:"uninjected,omitempty"`
}

// +kubebuilder:object:root=true
// +kubebuilder:resource:scope=Cluster
// +kubebuilder:subresource:status
// +kubebuilder:printcolumn:name="Control Plane Namespace",type=string,JSONPath=`.spec.global.linkerdNamespace`
// +kubebuilder:printcolumn:name="ConfigMap",type=string,JSONPath=`.spec.global.configMap`

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
	ClusterDomain          string          `json:"clusterDomain"`
	CNIEnabled             bool            `json:"cniEnabled"`
	ConfigMap              string          `json:"configMap"`
	IdentityContext        IdentityContext `json:"identityContext"`
	LinkerdNamespace       string          `json:"linkerdNamespace"`
	OmitWebhookSideEffects bool            `json:"omitWebhookSideEffects"`
	Version                string          `json:"version"`
}

// ToProtoString converts Global to the Linkerd protobuf format.
func (g *Global) ToProtoJSON() (string, error) {
	issuanceLifeTime, err := time.ParseDuration(g.IdentityContext.IssuanceLifeTime)
	if err != nil {
		return "", err
	}

	clockSkewAllowance, err := time.ParseDuration(g.IdentityContext.ClockSkewAllowance)
	if err != nil {
		return "", err
	}

	protoGlobal := &proto.Global{
		LinkerdNamespace: g.LinkerdNamespace,
		CniEnabled:       g.CNIEnabled,
		Version:          g.Version,
		IdentityContext: &proto.IdentityContext{
			TrustDomain:        g.IdentityContext.TrustDomain,
			TrustAnchorsPem:    g.IdentityContext.TrustAnchorsPEM,
			IssuanceLifetime:   ptypes.DurationProto(issuanceLifeTime),
			ClockSkewAllowance: ptypes.DurationProto(clockSkewAllowance),
			Scheme:             g.IdentityContext.Scheme,
		},
		OmitWebhookSideEffects: g.OmitWebhookSideEffects,
		ClusterDomain:          g.ClusterDomain,
	}

	marshaler := jsonpb.Marshaler{EmitDefaults: true}
	return marshaler.MarshalToString(protoGlobal)
}

// IdentityContext contains mTLS trust configuration.
type IdentityContext struct {
	TrustDomain        string `json:"trustDomain"`
	TrustAnchorsPEM    string `json:"trustAnchorsPEM"`
	IssuanceLifeTime   string `json:"issuanceLifeTime"`
	ClockSkewAllowance string `json:"clockSkewAllowance"`
	Scheme             string `json:"scheme"`
}

// Proxy contains the dataplane's proxy configuration.
type Proxy struct {
	AdminPort               Port      `json:"adminPort"`
	ControlPort             Port      `json:"controlPort"`
	DisableExternalProfiles bool      `json:"disableExternalProfiles"`
	InboundPort             Port      `json:"inboundPort"`
	LogLevel                string    `json:"logLevel"`
	OutboundPort            Port      `json:"outboundPort"`
	ProxyImage              Image     `json:"proxyImage"`
	ProxyInitImage          Image     `json:"proxyInitImage"`
	ProxyInitImageVersion   string    `json:"proxyInitImageVersion"`
	ProxyVersion            string    `json:"proxyVersion"`
	ProxyUID                int64     `json:"proxyUID"`
	Resource                Resources `json:"resource"`

	//+kubebuilder:validation:Optional
	IgnoreInboundPorts []PortRange `json:"ignoreInboundPorts,omitEmpty"`

	//+kubebuilder:validation:Optional
	IgnoreOutboundPorts []PortRange `json:"ignoreOutboundPorts,omitempty"`
}

// ToProtoString converts Global to the Linkerd protobuf format.
func (p *Proxy) ToProtoJSON() (string, error) {
	var ignoreInboundPorts []*proto.PortRange
	for _, ignore := range p.IgnoreInboundPorts {
		ignoreInboundPorts = append(ignoreInboundPorts, &proto.PortRange{PortRange: string(ignore)})
	}

	var ignoreOutboundPorts []*proto.PortRange
	for _, ignore := range p.IgnoreOutboundPorts {
		ignoreOutboundPorts = append(ignoreOutboundPorts, &proto.PortRange{PortRange: string(ignore)})
	}

	protoProxy := &proto.Proxy{
		ProxyImage: &proto.Image{
			ImageName:  p.ProxyImage.Name,
			PullPolicy: p.ProxyImage.PullPolicy,
		},
		ProxyInitImage: &proto.Image{
			ImageName:  p.ProxyInitImage.Name,
			PullPolicy: p.ProxyInitImage.PullPolicy,
		},
		ControlPort: &proto.Port{
			Port: uint32(p.ControlPort),
		},
		IgnoreInboundPorts:  ignoreInboundPorts,
		IgnoreOutboundPorts: ignoreOutboundPorts,
		InboundPort: &proto.Port{
			Port: uint32(p.InboundPort),
		},
		AdminPort: &proto.Port{
			Port: uint32(p.AdminPort),
		},
		OutboundPort: &proto.Port{
			Port: uint32(p.OutboundPort),
		},
		Resource: &proto.ResourceRequirements{
			RequestCpu:    "",
			RequestMemory: "",
			LimitCpu:      "",
			LimitMemory:   "",
		},
		ProxyUid: p.ProxyUID,
		LogLevel: &proto.LogLevel{
			Level: p.LogLevel,
		},
		DisableExternalProfiles: p.DisableExternalProfiles,
		ProxyVersion:            p.ProxyVersion,
		ProxyInitImageVersion:   p.ProxyInitImageVersion,
	}

	marshaler := jsonpb.Marshaler{EmitDefaults: true}
	return marshaler.MarshalToString(protoProxy)
}

// Image represents a container's image.
type Image struct {
	Name       string `json:"name"`
	PullPolicy string `json:"pullPolicy"`
}

// Port represents a port number.
type Port uint32

var emptyPort = Port(0)

func (p Port) empty() bool {
	return p == emptyPort
}

// PortRange is a range of ports.
type PortRange string

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
