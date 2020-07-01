package controllers

// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets,verbs=get;list;watch
// +kubebuilder:rbac:groups=policy,resources=poddisruptionbudgets/status,verbs=get

// PodDisruptionBudgetReconciler reconciles PodDisruptionBudgets
type PodDisruptionBudgetReconciler struct {
	ResourceReconciler
}
