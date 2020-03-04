package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	MinCheckInterval        = 10 * time.Second
	RequeueIntervalForError = 30 * time.Second
)

// EventFilter determine what events we care about. Kubernetes first filter events then hands off those events to EventHandler
type EventFilter struct {
	Log               logr.Logger
	ObjectGenerations map[string]int64
}

func (f EventFilter) CreateEventFilter(e event.CreateEvent) bool {
	return true
}

func (f EventFilter) DeleteEventFilter(e event.DeleteEvent) bool {
	return true
}

func (f EventFilter) UpdateEventFilter(e event.UpdateEvent) bool {
	f.Log.Info("event filter received update event",
		"name", e.MetaNew.GetName(),
		"old generation", e.MetaOld.GetGeneration(),
		"old resource version", e.MetaOld.GetResourceVersion(),
		"new generation", e.MetaNew.GetGeneration(),
		"new resource version", e.MetaNew.GetResourceVersion(),
		"generations cache", f.ObjectGenerations[e.MetaNew.GetName()],
	)
	gen, ok := f.ObjectGenerations[e.MetaNew.GetName()]
	if ok && e.MetaNew.GetGeneration() == gen {
		f.Log.Info("same generation as ObjectGenerations, event is from status updates.")
		return false
	}

	// if new generation == 0: resource is tracked by version, such as hpa
	// if not: resource can be tracked via generation, such as rule, so we need to check object generation
	return e.MetaNew.GetResourceVersion() != e.MetaOld.GetResourceVersion() || e.MetaNew.GetGeneration() != e.MetaOld.GetGeneration()
}

func (f EventFilter) GenericEventFilter(e event.GenericEvent) bool {
	return true
}

func GetPredicateFuncs(Log logr.Logger, generations map[string]int64) predicate.Funcs {
	e := EventFilter{Log: Log, ObjectGenerations: generations}
	return predicate.Funcs{
		CreateFunc:  e.CreateEventFilter,
		DeleteFunc:  e.DeleteEventFilter,
		UpdateFunc:  e.UpdateEventFilter,
		GenericFunc: e.GenericEventFilter,
	}
}
