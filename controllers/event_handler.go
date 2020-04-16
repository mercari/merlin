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

	if evt.MetaOld != nil {
		e.Log.Info("adding event to queue since old meta is not nil")
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      e.Kind + Separator + evt.MetaOld.GetName(),
			Namespace: evt.MetaOld.GetNamespace(),
		}})
		return
	} else {
		e.Log.Error(nil, "UpdateEvent received with no old metadata", "event", evt)
	}

	if evt.MetaNew != nil {
		e.Log.Info("adding event to queue since new meta is not nil")
		q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
			Name:      e.Kind + Separator + evt.MetaNew.GetName(),
			Namespace: evt.MetaNew.GetNamespace(),
		}})
		return
	} else {
		e.Log.Error(nil, "UpdateEvent received with no new metadata", "event", evt)
	}
}

// Create handles events from creating resources
func (e *EventHandler) Create(evt event.CreateEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received create event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "CreateEvent received with no metadata", "event", evt)
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      e.Kind + Separator + evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}})
}

// Delete handles events from deleting resources
func (e *EventHandler) Delete(evt event.DeleteEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received delete event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "DeleteEvent received with no metadata", "event", evt)
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      e.Kind + Separator + evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}})
}

// Generic handles events from generic operations
func (e *EventHandler) Generic(evt event.GenericEvent, q workqueue.RateLimitingInterface) {
	e.Log.V(1).Info("event handler received generic event", "name", evt.Meta.GetName())
	if evt.Meta == nil {
		e.Log.Error(nil, "GenericEvent received with no metadata", "event", evt)
		return
	}
	q.Add(reconcile.Request{NamespacedName: types.NamespacedName{
		Name:      e.Kind + Separator + evt.Meta.GetName(),
		Namespace: evt.Meta.GetNamespace(),
	}})
}
