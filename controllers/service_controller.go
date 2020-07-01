package controllers

// +kubebuilder:rbac:groups=core,resources=service,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=service/status,verbs=get

// ServiceReconciler reconciles service and rules for service objects
type ServiceReconciler struct {
	ResourceReconciler
}
