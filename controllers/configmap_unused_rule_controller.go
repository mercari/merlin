package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=clusterruleconfigmapunused,verbs=get;list;watch

// ConfigMapUnusedRuleReconciler reconciles ClusterRuleConfigMapUnused
type ConfigMapUnusedRuleReconciler struct {
	RuleReconciler
}
