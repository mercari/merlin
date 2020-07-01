package controllers

// +kubebuilder:rbac:groups=core,resources=clusterrulepdbinvalidselector,verbs=get;list;watch

// PDBInvalidSelectorRuleReconciler reconciles clusterrulepdbinvalidselector
type PDBInvalidSelectorRuleReconciler struct {
	RuleReconciler
}
