package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=configmaps,verbs=get;list;watch
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=configmaps/status,verbs=get

// ConfigMapReconciler reconciles configmap and rules for configmap objects
type ConfigMapReconciler struct {
	ResourceReconciler
}
