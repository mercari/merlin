package controllers

import (
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/workqueue"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

// EventHandler determine how events should be handled. Kubernetes first uses EventFilter then hands off those events to this handler
type EventHandler struct {
	// logger
	Log logr.Logger
	// Kind is object kind which associated with the event
	Kind string
}

const Separator = string(types.Separator)

// Update handles events from updating resources
func (e *EventHandler) Update(evt event.UpdateEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received update event",
		"name", evt.MetaNew.GetName(),
		"old generation", evt.MetaOld.GetGeneration(),
		"old resource version", evt.MetaOld.GetResourceVersion(),
		"new generation", evt.MetaNew.GetGeneration(),
		"new resource version", evt.MetaNew.GetResourceVersion(),
	)

	var req reconcile.Request
	if evt.MetaOld != nil {
		req = reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      evt.MetaOld.GetName(),
			Namespace: evt.MetaOld.GetNamespace(),
		}}

		e.Log.Info("adding event to queue since old meta is not nil")
	} else if evt.MetaNew != nil {
		req = reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      evt.MetaNew.GetName(),
			Namespace: evt.MetaNew.GetNamespace(),
		}}
		e.Log.Info("adding event to queue since new meta is not nil")
	} else {
		e.Log.Error(nil, "UpdateEvent received with no new or old metadata", "event", evt)
	}
	if e.Kind != "" {
		req.Name = e.Kind + Separator + req.Name
	}
	q.Add(req)
	return
}

// Create handles events from creating resources
func (e *EventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received create event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	if e.Kind != "" {
		req.Name = e.Kind + Separator + req.Name
	}
	q.Add(req)
}

// Delete handles events from deleting resources
func (e *EventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received delete event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}
	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	if e.Kind != "" {
		req.Name = e.Kind + Separator + req.Name
	}
	q.Add(req)
}

// Generic handles events from generic operations
func (e *EventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received generic event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}
	req := reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}}
	if e.Kind != "" {
		req.Name = e.Kind + Separator + req.Name
	}
	q.Add(req)
}
