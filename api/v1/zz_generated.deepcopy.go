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
	runtime "k8s.io/apimachinery/pkg/runtime"
)

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Canary) DeepCopyInto(out *Canary) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Canary.
func (in *Canary) DeepCopy() *Canary {
	if in == nil {
		return nil
	}
	out := new(Canary)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentEvaluator) DeepCopyInto(out *DeploymentEvaluator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentEvaluator.
func (in *DeploymentEvaluator) DeepCopy() *DeploymentEvaluator {
	if in == nil {
		return nil
	}
	out := new(DeploymentEvaluator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DeploymentEvaluator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentEvaluatorList) DeepCopyInto(out *DeploymentEvaluatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]DeploymentEvaluator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentEvaluatorList.
func (in *DeploymentEvaluatorList) DeepCopy() *DeploymentEvaluatorList {
	if in == nil {
		return nil
	}
	out := new(DeploymentEvaluatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *DeploymentEvaluatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentEvaluatorSpec) DeepCopyInto(out *DeploymentEvaluatorSpec) {
	*out = *in
	out.Canary = in.Canary
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentEvaluatorSpec.
func (in *DeploymentEvaluatorSpec) DeepCopy() *DeploymentEvaluatorSpec {
	if in == nil {
		return nil
	}
	out := new(DeploymentEvaluatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *DeploymentEvaluatorStatus) DeepCopyInto(out *DeploymentEvaluatorStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new DeploymentEvaluatorStatus.
func (in *DeploymentEvaluatorStatus) DeepCopy() *DeploymentEvaluatorStatus {
	if in == nil {
		return nil
	}
	out := new(DeploymentEvaluatorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HPAEvaluator) DeepCopyInto(out *HPAEvaluator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HPAEvaluator.
func (in *HPAEvaluator) DeepCopy() *HPAEvaluator {
	if in == nil {
		return nil
	}
	out := new(HPAEvaluator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HPAEvaluator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HPAEvaluatorList) DeepCopyInto(out *HPAEvaluatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]HPAEvaluator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HPAEvaluatorList.
func (in *HPAEvaluatorList) DeepCopy() *HPAEvaluatorList {
	if in == nil {
		return nil
	}
	out := new(HPAEvaluatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *HPAEvaluatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HPAEvaluatorSpec) DeepCopyInto(out *HPAEvaluatorSpec) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HPAEvaluatorSpec.
func (in *HPAEvaluatorSpec) DeepCopy() *HPAEvaluatorSpec {
	if in == nil {
		return nil
	}
	out := new(HPAEvaluatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *HPAEvaluatorStatus) DeepCopyInto(out *HPAEvaluatorStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new HPAEvaluatorStatus.
func (in *HPAEvaluatorStatus) DeepCopy() *HPAEvaluatorStatus {
	if in == nil {
		return nil
	}
	out := new(HPAEvaluatorStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *Notifiers) DeepCopyInto(out *Notifiers) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	out.Spec = in.Spec
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new Notifiers.
func (in *Notifiers) DeepCopy() *Notifiers {
	if in == nil {
		return nil
	}
	out := new(Notifiers)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *Notifiers) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifiersList) DeepCopyInto(out *NotifiersList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]Notifiers, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifiersList.
func (in *NotifiersList) DeepCopy() *NotifiersList {
	if in == nil {
		return nil
	}
	out := new(NotifiersList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *NotifiersList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifiersSpec) DeepCopyInto(out *NotifiersSpec) {
	*out = *in
	out.Slack = in.Slack
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifiersSpec.
func (in *NotifiersSpec) DeepCopy() *NotifiersSpec {
	if in == nil {
		return nil
	}
	out := new(NotifiersSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *NotifiersStatus) DeepCopyInto(out *NotifiersStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new NotifiersStatus.
func (in *NotifiersStatus) DeepCopy() *NotifiersStatus {
	if in == nil {
		return nil
	}
	out := new(NotifiersStatus)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodEvaluator) DeepCopyInto(out *PodEvaluator) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ObjectMeta.DeepCopyInto(&out.ObjectMeta)
	in.Spec.DeepCopyInto(&out.Spec)
	out.Status = in.Status
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodEvaluator.
func (in *PodEvaluator) DeepCopy() *PodEvaluator {
	if in == nil {
		return nil
	}
	out := new(PodEvaluator)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PodEvaluator) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodEvaluatorList) DeepCopyInto(out *PodEvaluatorList) {
	*out = *in
	out.TypeMeta = in.TypeMeta
	in.ListMeta.DeepCopyInto(&out.ListMeta)
	if in.Items != nil {
		in, out := &in.Items, &out.Items
		*out = make([]PodEvaluator, len(*in))
		for i := range *in {
			(*in)[i].DeepCopyInto(&(*out)[i])
		}
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodEvaluatorList.
func (in *PodEvaluatorList) DeepCopy() *PodEvaluatorList {
	if in == nil {
		return nil
	}
	out := new(PodEvaluatorList)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyObject is an autogenerated deepcopy function, copying the receiver, creating a new runtime.Object.
func (in *PodEvaluatorList) DeepCopyObject() runtime.Object {
	if c := in.DeepCopy(); c != nil {
		return c
	}
	return nil
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodEvaluatorSpec) DeepCopyInto(out *PodEvaluatorSpec) {
	*out = *in
	if in.IgnoreNamespaces != nil {
		in, out := &in.IgnoreNamespaces, &out.IgnoreNamespaces
		*out = make([]string, len(*in))
		copy(*out, *in)
	}
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodEvaluatorSpec.
func (in *PodEvaluatorSpec) DeepCopy() *PodEvaluatorSpec {
	if in == nil {
		return nil
	}
	out := new(PodEvaluatorSpec)
	in.DeepCopyInto(out)
	return out
}

// DeepCopyInto is an autogenerated deepcopy function, copying the receiver, writing into out. in must be non-nil.
func (in *PodEvaluatorStatus) DeepCopyInto(out *PodEvaluatorStatus) {
	*out = *in
}

// DeepCopy is an autogenerated deepcopy function, copying the receiver, creating a new PodEvaluatorStatus.
func (in *PodEvaluatorStatus) DeepCopy() *PodEvaluatorStatus {
	if in == nil {
		return nil
	}
	out := new(PodEvaluatorStatus)
	in.DeepCopyInto(out)
	return out
}
