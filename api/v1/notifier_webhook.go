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
	"k8s.io/apimachinery/pkg/runtime"
	ctrl "sigs.k8s.io/controller-runtime"
	logf "sigs.k8s.io/controller-runtime/pkg/runtime/log"
	"sigs.k8s.io/controller-runtime/pkg/webhook"
)

// log is for logging in this package.
var notifierlog = logf.Log.WithName("notifier-resource")

func (r *Notifier) SetupWebhookWithManager(mgr ctrl.Manager) error {
	return ctrl.NewWebhookManagedBy(mgr).
		For(r).
		Complete()
}

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!

// +kubebuilder:webhook:path=/mutate-watcher-merlin-mercari-com-v1-notifier,mutating=true,failurePolicy=fail,groups=watcher.merlin.mercari.com,resources=notifiers,verbs=create;update,versions=v1,name=mnotifier.kb.io

var _ webhook.Defaulter = &Notifier{}

// Default implements webhook.Defaulter so a webhook will be registered for the type
func (r *Notifier) Default() {
	notifierlog.Info("default", "name", r.Name)

	// TODO(user): fill in your defaulting logic.
}

// TODO(user): change verbs to "verbs=create;update;delete" if you want to enable deletion validation.
// +kubebuilder:webhook:verbs=create;update,path=/validate-watcher-merlin-mercari-com-v1-notifier,mutating=false,failurePolicy=fail,groups=watcher.merlin.mercari.com,resources=notifiers,versions=v1,name=vnotifier.kb.io

var _ webhook.Validator = &Notifier{}

// ValidateCreate implements webhook.Validator so a webhook will be registered for the type
func (r *Notifier) ValidateCreate() error {
	notifierlog.Info("validate create", "name", r.Name)

	// TODO(user): fill in your validation logic upon object creation.
	return nil
}

// ValidateUpdate implements webhook.Validator so a webhook will be registered for the type
func (r *Notifier) ValidateUpdate(old runtime.Object) error {
	notifierlog.Info("validate update", "name", r.Name)

	// TODO(user): fill in your validation logic upon object update.
	return nil
}

// ValidateDelete implements webhook.Validator so a webhook will be registered for the type
func (r *Notifier) ValidateDelete() error {
	notifierlog.Info("validate delete", "name", r.Name)

	// TODO(user): fill in your validation logic upon object deletion.
	return nil
}
