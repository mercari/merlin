package controllers

// +kubebuilder:rbac:groups=core,resources=namespaces,verbs=get;list;watch
// +kubebuilder:rbac:groups=core,resources=namespaces/status,verbs=get

// NamespaceReconciler reconciles a Namespace object
type NamespaceReconciler struct {
	ResourceReconciler
}
