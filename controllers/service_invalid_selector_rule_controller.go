package controllers

// +kubebuilder:rbac:groups=core,resources=clusterruleserviceinvalidselector,verbs=get;list;watch

// ServiceInvalidSelectorRuleReconciler reconciles rules for ClusterRuleServiceInvalidSelector
type ServiceInvalidSelectorRuleReconciler struct {
	RuleReconciler
}
