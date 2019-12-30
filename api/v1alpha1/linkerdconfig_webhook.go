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
	"k8s.io/apimachinery/pkg/runtime"
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

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-config-linkerd-io-v1alpha1-linkerdconfig,mutating=false,failurePolicy=fail,groups=config.linkerd.io,resources=linkerdconfigs,versions=v1alpha1,name=linkerd-config.linkerd.io

var _ webhook.Validator = &LinkerdConfig{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateCreate() error {
	linkerdconfiglog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateUpdate(old runtime.Object) error {
	linkerdconfiglog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *LinkerdConfig) ValidateDelete() error {
	linkerdconfiglog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
