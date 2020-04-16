package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
)

// EventFilter determine what events we care about. Kubernetes first filter events then hands off those events to EventHandler
type EventFilter struct {
	predicate.Funcs
	Log logr.Logger
}

func (e *EventFilter) Update(evt event.UpdateEvent) bool {
	e.Log.V(1).Info("event filter received update event",
		"name", evt.MetaNew.GetName(),
		"old generation", evt.MetaOld.GetGeneration(),
		"old resource version", evt.MetaOld.GetResourceVersion(),
		"new generation", evt.MetaNew.GetGeneration(),
		"new resource version", evt.MetaNew.GetResourceVersion(),
	)
	// if new generation == 0: resource is tracked by version, such as hpa
	// if not: resource can be tracked via generation, such as rule, so we need to check object generation
	if evt.MetaNew.GetGeneration() == 0 {
		return evt.MetaNew.GetResourceVersion() != evt.MetaOld.GetResourceVersion()
	}
	return evt.MetaNew.GetGeneration() != evt.MetaOld.GetGeneration()
}

func (e *EventFilter) Create(evt event.CreateEvent) bool {
	e.Log.V(1).Info("event filter received create event", "name", evt.Meta.GetName())
	return true
}

func (e *EventFilter) Delete(evt event.DeleteEvent) bool {
	e.Log.V(1).Info("event filter received delete event", "name", evt.Meta.GetName())
	return true
}

func (e *EventFilter) Generic(evt event.GenericEvent) bool {
	e.Log.V(1).Info("event filter received generic event", "name", evt.Meta.GetName())
	return true
}
