package controllers

// +kubebuilder:rbac:groups=merlin.mercari.com,resources=secrets,verbs=get;list;watch
// +kubebuilder:rbac:groups=merlin.mercari.com,resources=secrets/status,verbs=get

// SecretReconciler reconciles secret and rules for secret objects
type SecretReconciler struct {
	ResourceReconciler
}
