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

package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=clusterrulehpareplicapercentage,verbs=get;list;watch
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=rulehpareplicapercentage,verbs=get;list;watch

// HPAReplicaPercentageRuleReconciler reconciles rule of secret unused
type HPAReplicaPercentageRuleReconciler struct {
	RuleReconciler
}
