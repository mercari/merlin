// +build !ignore_autogenerated

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

// Code generated by controller-gen. DO NOT EDIT.

package v1

import (
	"github.com/kouzoh/merlin/notifiers/alert"
	"k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAInvalidScaleTargetRef) DeepCopyInto(out *ClusterRuleHPAInvalidScaleTargetRef) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAInvalidScaleTargetRef.
func (in *ClusterRuleHPAInvalidScaleTargetRef) DeepCopy() *ClusterRuleHPAInvalidScaleTargetRef {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAInvalidScaleTargetRef)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleHPAInvalidScaleTargetRef) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAInvalidScaleTargetRefList) DeepCopyInto(out *ClusterRuleHPAInvalidScaleTargetRefList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRuleHPAInvalidScaleTargetRef, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAInvalidScaleTargetRefList.
func (in *ClusterRuleHPAInvalidScaleTargetRefList) DeepCopy() *ClusterRuleHPAInvalidScaleTargetRefList {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAInvalidScaleTargetRefList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleHPAInvalidScaleTargetRefList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAInvalidScaleTargetRefSpec) DeepCopyInto(out *ClusterRuleHPAInvalidScaleTargetRefSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAInvalidScaleTargetRefSpec.
func (in *ClusterRuleHPAInvalidScaleTargetRefSpec) DeepCopy() *ClusterRuleHPAInvalidScaleTargetRefSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAInvalidScaleTargetRefSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAReplicaPercentage) DeepCopyInto(out *ClusterRuleHPAReplicaPercentage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAReplicaPercentage.
func (in *ClusterRuleHPAReplicaPercentage) DeepCopy() *ClusterRuleHPAReplicaPercentage {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAReplicaPercentage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleHPAReplicaPercentage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAReplicaPercentageList) DeepCopyInto(out *ClusterRuleHPAReplicaPercentageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRuleHPAReplicaPercentage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAReplicaPercentageList.
func (in *ClusterRuleHPAReplicaPercentageList) DeepCopy() *ClusterRuleHPAReplicaPercentageList {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAReplicaPercentageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleHPAReplicaPercentageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleHPAReplicaPercentageSpec) DeepCopyInto(out *ClusterRuleHPAReplicaPercentageSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleHPAReplicaPercentageSpec.
func (in *ClusterRuleHPAReplicaPercentageSpec) DeepCopy() *ClusterRuleHPAReplicaPercentageSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleHPAReplicaPercentageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleNamespaceRequiredLabel) DeepCopyInto(out *ClusterRuleNamespaceRequiredLabel) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleNamespaceRequiredLabel.
func (in *ClusterRuleNamespaceRequiredLabel) DeepCopy() *ClusterRuleNamespaceRequiredLabel {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleNamespaceRequiredLabel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleNamespaceRequiredLabel) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleNamespaceRequiredLabelList) DeepCopyInto(out *ClusterRuleNamespaceRequiredLabelList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRuleNamespaceRequiredLabel, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleNamespaceRequiredLabelList.
func (in *ClusterRuleNamespaceRequiredLabelList) DeepCopy() *ClusterRuleNamespaceRequiredLabelList {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleNamespaceRequiredLabelList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleNamespaceRequiredLabelList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleNamespaceRequiredLabelSpec) DeepCopyInto(out *ClusterRuleNamespaceRequiredLabelSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
	out.Label = in.Label
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleNamespaceRequiredLabelSpec.
func (in *ClusterRuleNamespaceRequiredLabelSpec) DeepCopy() *ClusterRuleNamespaceRequiredLabelSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleNamespaceRequiredLabelSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBInvalidSelector) DeepCopyInto(out *ClusterRulePDBInvalidSelector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBInvalidSelector.
func (in *ClusterRulePDBInvalidSelector) DeepCopy() *ClusterRulePDBInvalidSelector {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBInvalidSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRulePDBInvalidSelector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBInvalidSelectorList) DeepCopyInto(out *ClusterRulePDBInvalidSelectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRulePDBInvalidSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBInvalidSelectorList.
func (in *ClusterRulePDBInvalidSelectorList) DeepCopy() *ClusterRulePDBInvalidSelectorList {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBInvalidSelectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRulePDBInvalidSelectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBInvalidSelectorSpec) DeepCopyInto(out *ClusterRulePDBInvalidSelectorSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBInvalidSelectorSpec.
func (in *ClusterRulePDBInvalidSelectorSpec) DeepCopy() *ClusterRulePDBInvalidSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBInvalidSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBMinAllowedDisruption) DeepCopyInto(out *ClusterRulePDBMinAllowedDisruption) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBMinAllowedDisruption.
func (in *ClusterRulePDBMinAllowedDisruption) DeepCopy() *ClusterRulePDBMinAllowedDisruption {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBMinAllowedDisruption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRulePDBMinAllowedDisruption) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBMinAllowedDisruptionList) DeepCopyInto(out *ClusterRulePDBMinAllowedDisruptionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRulePDBMinAllowedDisruption, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBMinAllowedDisruptionList.
func (in *ClusterRulePDBMinAllowedDisruptionList) DeepCopy() *ClusterRulePDBMinAllowedDisruptionList {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBMinAllowedDisruptionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRulePDBMinAllowedDisruptionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRulePDBMinAllowedDisruptionSpec) DeepCopyInto(out *ClusterRulePDBMinAllowedDisruptionSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRulePDBMinAllowedDisruptionSpec.
func (in *ClusterRulePDBMinAllowedDisruptionSpec) DeepCopy() *ClusterRulePDBMinAllowedDisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRulePDBMinAllowedDisruptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleServiceInvalidSelector) DeepCopyInto(out *ClusterRuleServiceInvalidSelector) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleServiceInvalidSelector.
func (in *ClusterRuleServiceInvalidSelector) DeepCopy() *ClusterRuleServiceInvalidSelector {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleServiceInvalidSelector)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleServiceInvalidSelector) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleServiceInvalidSelectorList) DeepCopyInto(out *ClusterRuleServiceInvalidSelectorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]ClusterRuleServiceInvalidSelector, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleServiceInvalidSelectorList.
func (in *ClusterRuleServiceInvalidSelectorList) DeepCopy() *ClusterRuleServiceInvalidSelectorList {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleServiceInvalidSelectorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *ClusterRuleServiceInvalidSelectorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *ClusterRuleServiceInvalidSelectorSpec) DeepCopyInto(out *ClusterRuleServiceInvalidSelectorSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
	in.Notification.DeepCopyInto(&out.Notification)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new ClusterRuleServiceInvalidSelectorSpec.
func (in *ClusterRuleServiceInvalidSelectorSpec) DeepCopy() *ClusterRuleServiceInvalidSelectorSpec {
	if in == nil {
		return nil
	}
	out := new(ClusterRuleServiceInvalidSelectorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Notification) DeepCopyInto(out *Notification) {
	*out = *in
	if in.Notifiers != nil {
		in, out := &in.Notifiers, &out.Notifiers
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Notification.
func (in *Notification) DeepCopy() *Notification {
	if in == nil {
		return nil
	}
	out := new(Notification)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Notifier) DeepCopyInto(out *Notifier) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Notifier.
func (in *Notifier) DeepCopy() *Notifier {
	if in == nil {
		return nil
	}
	out := new(Notifier)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Notifier) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifierList) DeepCopyInto(out *NotifierList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Notifier, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifierList.
func (in *NotifierList) DeepCopy() *NotifierList {
	if in == nil {
		return nil
	}
	out := new(NotifierList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NotifierList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifierSpec) DeepCopyInto(out *NotifierSpec) {
	*out = *in
	out.Slack = in.Slack
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifierSpec.
func (in *NotifierSpec) DeepCopy() *NotifierSpec {
	if in == nil {
		return nil
	}
	out := new(NotifierSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifierStatus) DeepCopyInto(out *NotifierStatus) {
	*out = *in
	if in.Alerts != nil {
		in, out := &in.Alerts, &out.Alerts
		*out = make(map[string]alert.Alert, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifierStatus.
func (in *NotifierStatus) DeepCopy() *NotifierStatus {
	if in == nil {
		return nil
	}
	out := new(NotifierStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifiersCache) DeepCopyInto(out *NotifiersCache) {
	*out = *in
	if in.Notifiers != nil {
		in, out := &in.Notifiers, &out.Notifiers
		*out = make(map[string]*Notifier, len(*in))
		for key, val := range *in {
			var outVal *Notifier
			if val == nil {
				(*out)[key] = nil
			} else {
				in, out := &val, &outVal
				*out = new(Notifier)
				(*in).DeepCopyInto(*out)
			}
			(*out)[key] = outVal
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifiersCache.
func (in *NotifiersCache) DeepCopy() *NotifiersCache {
	if in == nil {
		return nil
	}
	out := new(NotifiersCache)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RequiredLabel) DeepCopyInto(out *RequiredLabel) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RequiredLabel.
func (in *RequiredLabel) DeepCopy() *RequiredLabel {
	if in == nil {
		return nil
	}
	out := new(RequiredLabel)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuleHPAReplicaPercentage) DeepCopyInto(out *RuleHPAReplicaPercentage) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuleHPAReplicaPercentage.
func (in *RuleHPAReplicaPercentage) DeepCopy() *RuleHPAReplicaPercentage {
	if in == nil {
		return nil
	}
	out := new(RuleHPAReplicaPercentage)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RuleHPAReplicaPercentage) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuleHPAReplicaPercentageList) DeepCopyInto(out *RuleHPAReplicaPercentageList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RuleHPAReplicaPercentage, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuleHPAReplicaPercentageList.
func (in *RuleHPAReplicaPercentageList) DeepCopy() *RuleHPAReplicaPercentageList {
	if in == nil {
		return nil
	}
	out := new(RuleHPAReplicaPercentageList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RuleHPAReplicaPercentageList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuleHPAReplicaPercentageSpec) DeepCopyInto(out *RuleHPAReplicaPercentageSpec) {
	*out = *in
	in.Notification.DeepCopyInto(&out.Notification)
	in.Selector.DeepCopyInto(&out.Selector)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuleHPAReplicaPercentageSpec.
func (in *RuleHPAReplicaPercentageSpec) DeepCopy() *RuleHPAReplicaPercentageSpec {
	if in == nil {
		return nil
	}
	out := new(RuleHPAReplicaPercentageSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePDBMinAllowedDisruption) DeepCopyInto(out *RulePDBMinAllowedDisruption) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	in.Status.DeepCopyInto(&out.Status)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePDBMinAllowedDisruption.
func (in *RulePDBMinAllowedDisruption) DeepCopy() *RulePDBMinAllowedDisruption {
	if in == nil {
		return nil
	}
	out := new(RulePDBMinAllowedDisruption)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RulePDBMinAllowedDisruption) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePDBMinAllowedDisruptionList) DeepCopyInto(out *RulePDBMinAllowedDisruptionList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RulePDBMinAllowedDisruption, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePDBMinAllowedDisruptionList.
func (in *RulePDBMinAllowedDisruptionList) DeepCopy() *RulePDBMinAllowedDisruptionList {
	if in == nil {
		return nil
	}
	out := new(RulePDBMinAllowedDisruptionList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RulePDBMinAllowedDisruptionList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePDBMinAllowedDisruptionSpec) DeepCopyInto(out *RulePDBMinAllowedDisruptionSpec) {
	*out = *in
	in.Notification.DeepCopyInto(&out.Notification)
	in.Selector.DeepCopyInto(&out.Selector)
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePDBMinAllowedDisruptionSpec.
func (in *RulePDBMinAllowedDisruptionSpec) DeepCopy() *RulePDBMinAllowedDisruptionSpec {
	if in == nil {
		return nil
	}
	out := new(RulePDBMinAllowedDisruptionSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePodResources) DeepCopyInto(out *RulePodResources) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePodResources.
func (in *RulePodResources) DeepCopy() *RulePodResources {
	if in == nil {
		return nil
	}
	out := new(RulePodResources)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RulePodResources) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePodResourcesList) DeepCopyInto(out *RulePodResourcesList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]RulePodResources, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePodResourcesList.
func (in *RulePodResourcesList) DeepCopy() *RulePodResourcesList {
	if in == nil {
		return nil
	}
	out := new(RulePodResourcesList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *RulePodResourcesList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePodResourcesSpec) DeepCopyInto(out *RulePodResourcesSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePodResourcesSpec.
func (in *RulePodResourcesSpec) DeepCopy() *RulePodResourcesSpec {
	if in == nil {
		return nil
	}
	out := new(RulePodResourcesSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RulePodResourcesStatus) DeepCopyInto(out *RulePodResourcesStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RulePodResourcesStatus.
func (in *RulePodResourcesStatus) DeepCopy() *RulePodResourcesStatus {
	if in == nil {
		return nil
	}
	out := new(RulePodResourcesStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *RuleStatus) DeepCopyInto(out *RuleStatus) {
	*out = *in
	if in.Violations != nil {
		in, out := &in.Violations, &out.Violations
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new RuleStatus.
func (in *RuleStatus) DeepCopy() *RuleStatus {
	if in == nil {
		return nil
	}
	out := new(RuleStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Selector) DeepCopyInto(out *Selector) {
	*out = *in
	if in.MatchLabels != nil {
		in, out := &in.MatchLabels, &out.MatchLabels
		*out = make(map[string]string, len(*in))
		for key, val := range *in {
			(*out)[key] = val
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Selector.
func (in *Selector) DeepCopy() *Selector {
	if in == nil {
		return nil
	}
	out := new(Selector)
	in.DeepCopyInto(out)
	return out
}
