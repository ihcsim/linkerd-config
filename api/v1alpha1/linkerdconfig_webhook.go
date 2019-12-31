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
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/util/validation/field"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var linkerdconfiglog = logf.Log.WithName("linkerdconfig-resource")

func (r *LinkerdConfig) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-config-linkerd-io-v1alpha1-linkerdconfig,mutating=true,failurePolicy=fail,groups=config.linkerd.io,resources=linkerdconfigs,verbs=create;update,versions=v1alpha1,name=linkerd-config.linkerd.io

var _ webhook.Defaulter = &LinkerdConfig{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *LinkerdConfig) Default() {
	linkerdconfiglog.Info("default", "name", r.Name)

	if r.Spec.Global.ClusterDomain == "" {
		r.Spec.Global.ClusterDomain = "cluster.local"
	}

	if r.Spec.Global.ConfigMap == "" {
		r.Spec.Global.ConfigMap = "linkerd-config"
	}

	if r.Spec.Global.IdentityContext.TrustDomain == "" {
		r.Spec.Global.IdentityContext.TrustDomain = "cluster.local"
	}

	if r.Spec.Global.IdentityContext.IssuanceLifeTime == "" {
		r.Spec.Global.IdentityContext.IssuanceLifeTime = "86400s"
	}

	if r.Spec.Global.IdentityContext.ClockSkewAllowance == "" {
		r.Spec.Global.IdentityContext.ClockSkewAllowance = "20s"
	}

	if r.Spec.Global.IdentityContext.Scheme == "" {
		r.Spec.Global.IdentityContext.Scheme = "linkerd.io/tls"
	}

	if r.Spec.Global.LinkerdNamespace == "" {
		r.Spec.Global.LinkerdNamespace = "linkerd"
	}

	if r.Spec.Proxy.AdminPort.empty() {
		r.Spec.Proxy.AdminPort = Port(4191)
	}

	if r.Spec.Proxy.ControlPort.empty() {
		r.Spec.Proxy.ControlPort = Port(4190)
	}

	if r.Spec.Proxy.InboundPort.empty() {
		r.Spec.Proxy.InboundPort = Port(4143)
	}

	if r.Spec.Proxy.LogLevel == "" {
		r.Spec.Proxy.LogLevel = "warn,linkerd2_proxy=info"
	}

	if r.Spec.Proxy.OutboundPort.empty() {
		r.Spec.Proxy.OutboundPort = Port(4140)
	}

	if r.Spec.Proxy.ProxyImage.Name == "" {
		r.Spec.Proxy.ProxyImage.Name = "gcr.io/linkerd-io/proxy"
	}

	if r.Spec.Proxy.ProxyImage.PullPolicy == "" {
		r.Spec.Proxy.ProxyImage.Name = "IfNotPresent"
	}

	if r.Spec.Proxy.ProxyInitImage.Name == "" {
		r.Spec.Proxy.ProxyInitImage.Name = "gcr.io/linkerd-io/proxy"
	}

	if r.Spec.Proxy.ProxyInitImage.PullPolicy == "" {
		r.Spec.Proxy.ProxyInitImage.Name = "IfNotPresent"
	}

	if r.Spec.Proxy.ProxyUID == 0 {
		r.Spec.Proxy.ProxyUID = 2102
	}

	if r.Spec.Proxy.Resource.Limits == nil {
		r.Spec.Proxy.Resource.Limits = map[string]string{}
	}

	if r.Spec.Proxy.Resource.Limits["cpu"] == "" {
		r.Spec.Proxy.Resource.Limits["cpu"] = "1"
	}

	if r.Spec.Proxy.Resource.Limits["memory"] == "" {
		r.Spec.Proxy.Resource.Limits["memory"] = "250Mi"
	}

	if r.Spec.Proxy.Resource.Requests == nil {
		r.Spec.Proxy.Resource.Requests = map[string]string{}
	}

	if r.Spec.Proxy.Resource.Requests["cpu"] == "" {
		r.Spec.Proxy.Resource.Requests["cpu"] = "100m"
	}

	if r.Spec.Proxy.Resource.Requests["memory"] == "" {
		r.Spec.Proxy.Resource.Requests["memory"] = "20Mi"
	}

	if r.Spec.Proxy.IgnoreInboundPorts == nil {
		r.Spec.Proxy.IgnoreInboundPorts = Ports{}
	}

	if r.Spec.Proxy.IgnoreOutboundPorts == nil {
		r.Spec.Proxy.IgnoreOutboundPorts = Ports{}
	}
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-config-linkerd-io-v1alpha1-linkerdconfig,mutating=false,failurePolicy=fail,groups=config.linkerd.io,resources=linkerdconfigs,versions=v1alpha1,name=linkerd-config.linkerd.io

var _ webhook.Validator = &LinkerdConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateCreate() error {
	linkerdconfiglog.Info("validate create", "name", r.Name)
	return r.validate()
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateUpdate(old runtime.Object) error {
	linkerdconfiglog.Info("validate update", "name", r.Name)
	return r.validate()
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateDelete() error {
	linkerdconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}

func (r *LinkerdConfig) validate() error {
	var (
		errors field.ErrorList
		root   = field.NewPath("spec")
	)

	if r.Spec.Global.IdentityContext.TrustAnchorsPEM == "" {
		path := root.Child("global", "identityContext", "trustAnchorsPEM")
		errors = append(errors, field.Required(path, "trust anchors must be provided"))
	}

	if r.Spec.Proxy.ProxyVersion == "" {
		path := root.Child("proxy", "proxyVersion")
		errors = append(errors, field.Required(path, "proxy image version must be provided"))
	}

	if r.Spec.Proxy.ProxyInitImageVersion == "" {
		path := root.Child("proxy", "proxyInitImageVersion")
		errors = append(errors, field.Required(path, "proxy-init image version must be provided"))
	}

	if len(errors) > 0 {
		return apierrors.NewInvalid(
			schema.GroupKind{
				Group: GroupVersion.Group,
				Kind:  "LinkerdConfig"},
			r.Name, errors,
		)
	}

	return nil
}
