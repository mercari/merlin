package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"sync"
	"time"
)

const (
	RequeueIntervalForError = 30 * time.Second
)

// EventFilter determine what events we care about. Kubernetes first filter events then hands off those events to EventHandler
type EventFilter struct {
	Log               logr.Logger
	ObjectGenerations *sync.Map
}

func (e *EventFilter) CreateEventFilter(evt event.CreateEvent) bool {
	return true
}

func (e *EventFilter) DeleteEventFilter(evt event.DeleteEvent) bool {
	e.Log.Info("event filter received delete event")
	return true
}

func (e *EventFilter) UpdateEventFilter(evt event.UpdateEvent) bool {
	e.Log.Info("event filter received update event",
		"name", evt.MetaNew.GetName(),
		"old generation", evt.MetaOld.GetGeneration(),
		"old resource version", evt.MetaOld.GetResourceVersion(),
		"new generation", evt.MetaNew.GetGeneration(),
		"new resource version", evt.MetaNew.GetResourceVersion(),
	)
	gen, ok := e.ObjectGenerations.Load(evt.MetaNew.GetName())
	if ok && evt.MetaNew.GetGeneration() == gen {
		e.Log.Info("same generation as ObjectGenerations, event is from status updates.")
		return false
	}

	// if new generation == 0: resource is tracked by version, such as hpa
	// if not: resource can be tracked via generation, such as rule, so we need to check object generation
	return evt.MetaNew.GetResourceVersion() != evt.MetaOld.GetResourceVersion() || evt.MetaNew.GetGeneration() != evt.MetaOld.GetGeneration()
}

func (e *EventFilter) GenericEventFilter(evt event.GenericEvent) bool {
	return true
}

func GetPredicateFuncs(Log logr.Logger, generations *sync.Map) *predicate.Funcs {
	e := &EventFilter{Log: Log, ObjectGenerations: generations}
	return &predicate.Funcs{
		CreateFunc:  e.CreateEventFilter,
		DeleteFunc:  e.DeleteEventFilter,
		UpdateFunc:  e.UpdateEventFilter,
		GenericFunc: e.GenericEventFilter,
	}
}
