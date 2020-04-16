package v1

import (
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1beta1 "k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
)

// Here are the list of object list structs that we extend, they have List/AddItem/ListItems functions
// so that the Reconciler can be generic for all resources that it watches, if we want to support more
// resources then we need to add the new list.
// List returns the builtin resource list
// AddItem adds an object into Items
// ListItems returns the list of items

type policyv1beta1PDBList struct {
	policyv1beta1.PodDisruptionBudgetList
}

func (s *policyv1beta1PDBList) List() runtime.Object {
	return &s.PodDisruptionBudgetList
}

func (s *policyv1beta1PDBList) AddItem(object types.NamespacedName) {
	s.Items = append(s.Items, policyv1beta1.PodDisruptionBudget{ObjectMeta: metav1.ObjectMeta{Name: object.Name, Namespace: object.Namespace}})
}

func (s *policyv1beta1PDBList) ListItems() []interface{} {
	l := make([]interface{}, len(s.Items))
	for i, v := range s.Items {
		l[i] = v
	}
	return l
}

type coreV1ServiceList struct {
	corev1.ServiceList
}

func (s *coreV1ServiceList) List() runtime.Object {
	return &s.ServiceList
}

func (s *coreV1ServiceList) AddItem(object types.NamespacedName) {
	s.Items = append(s.Items, corev1.Service{ObjectMeta: metav1.ObjectMeta{Name: object.Name, Namespace: object.Namespace}})
}

func (s *coreV1ServiceList) ListItems() []interface{} {
	l := make([]interface{}, len(s.Items))
	for i, v := range s.Items {
		l[i] = v
	}
	return l
}

type coreV1NamespaceList struct {
	corev1.NamespaceList
}

func (s *coreV1NamespaceList) List() runtime.Object {
	return &s.NamespaceList
}

func (s *coreV1NamespaceList) AddItem(object types.NamespacedName) {
	s.Items = append(s.Items, corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: object.Name, Namespace: object.Namespace}})
}

func (s *coreV1NamespaceList) ListItems() []interface{} {
	l := make([]interface{}, len(s.Items))
	for i, v := range s.Items {
		l[i] = v
	}
	return l
}

type autoscalingv1HPAList struct {
	autoscalingv1.HorizontalPodAutoscalerList
}

func (s *autoscalingv1HPAList) List() runtime.Object {
	return &s.HorizontalPodAutoscalerList
}

func (s *autoscalingv1HPAList) AddItem(object types.NamespacedName) {
	s.Items = append(s.Items, autoscalingv1.HorizontalPodAutoscaler{ObjectMeta: metav1.ObjectMeta{Name: object.Name, Namespace: object.Namespace}})
}

func (s *autoscalingv1HPAList) ListItems() []interface{} {
	l := make([]interface{}, len(s.Items))
	for i, v := range s.Items {
		l[i] = v
	}
	return l
}
