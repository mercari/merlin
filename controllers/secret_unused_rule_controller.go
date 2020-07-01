package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=clusterrulesecretunused,verbs=get;list;watch

// SecretUnusedReconciler reconciles ClusterRuleSecretUnused
type SecretUnusedRuleReconciler struct {
	RuleReconciler
}
