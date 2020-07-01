package controllers

// +kubebuilder:rbac:groups=core,resources=pods,verbs=get;list;watch;create;update;patch;delete
// +kubebuilder:rbac:groups=core,resources=pods/status,verbs=get;update;patch

// PodReconciler reconciles a Pod object
type PodReconciler struct {
	ResourceReconciler
}
