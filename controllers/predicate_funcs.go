package controllers

import (
	"github.com/go-logr/logr"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	AnnotationCheckedTime = "merlin.mercari.com/checked-at"
	AnnotationIssue       = "merlin.mercari.com/issue"
)

type EventFilter struct {
	Log logr.Logger
}

func (f EventFilter) CreateEventFilter(e event.CreateEvent) bool {
	// prevents from massive event processing at startup time,
	// but if we have cache mechanism this might not be necessary
	_, ok := e.Meta.GetAnnotations()[AnnotationCheckedTime]
	return ok
}

func (f EventFilter) DeleteEventFilter(e event.DeleteEvent) bool {
	return false
}

func (f EventFilter) UpdateEventFilter(e event.UpdateEvent) bool {
	if lastChecked, ok := e.MetaNew.GetAnnotations()[AnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return true
		}

		return lastCheckedTime.Add(3 * time.Second).Before(time.Now())
	}
	return true
}

func (f EventFilter) GenericEventFilter(e event.GenericEvent) bool {
	if lastChecked, ok := e.Meta.GetAnnotations()[AnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return true
		}
		return lastCheckedTime.Add(3 * time.Second).After(time.Now())
	}
	return true
}

func GetPredicateFuncs(Log logr.Logger) predicate.Funcs {
	e := EventFilter{Log: Log}
	return predicate.Funcs{
		CreateFunc:  e.CreateEventFilter,
		DeleteFunc:  e.DeleteEventFilter,
		UpdateFunc:  e.UpdateEventFilter,
		GenericFunc: e.GenericEventFilter,
	}
}
