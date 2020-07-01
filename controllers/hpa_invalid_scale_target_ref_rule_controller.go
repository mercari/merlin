package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=clusterrulehpainvalidscaletargetref,verbs=get;list;watch

// HPAInvalidScaleTargetRefRuleReconciler reconciles rule of secret unused
type HPAInvalidScaleTargetRefRuleReconciler struct {
	RuleReconciler
}
