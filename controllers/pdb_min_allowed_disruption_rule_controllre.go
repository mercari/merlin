package controllers

// +kubebuilder:rbac:groups=core,resources=clusterrulepdbminalloweddisruption,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=rulepdbminalloweddisruption,verbs=get;list;watch

// PDBMinAllowedDisruptionRuleReconciler reconciles clusterrulepdbminalloweddisruption and rulepdbminalloweddisruption
type PDBMinAllowedDisruptionRuleReconciler struct {
	RuleReconciler
}
