package controllers

import (
	"github.com/go-logr/logr"
	merlinv1 "github.com/kouzoh/merlin/api/v1"
	"sigs.k8s.io/controller-runtime/pkg/event"
	"sigs.k8s.io/controller-runtime/pkg/predicate"
	"time"
)

const (
	AnnotationCheckedTime = merlinv1.GROUP + "/checked-at"
	AnnotationIssue       = merlinv1.GROUP + "/issue"
	MinCheckInterval      = 10 * time.Second
)

type EventFilter struct {
	Log logr.Logger
}

func (f EventFilter) CreateEventFilter(e event.CreateEvent) bool {
	if lastChecked, ok := e.Meta.GetAnnotations()[AnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return true
		}
		return lastCheckedTime.Add(MinCheckInterval).Before(time.Now())
	}
	return true
}

func (f EventFilter) DeleteEventFilter(e event.DeleteEvent) bool {
	return false
}

func (f EventFilter) UpdateEventFilter(e event.UpdateEvent) bool {
	if e.MetaOld.GetAnnotations()[AnnotationCheckedTime] != e.MetaNew.GetAnnotations()[AnnotationCheckedTime] {
		return false // annotation change, no need to process again.
	}
	if lastChecked, ok := e.MetaNew.GetAnnotations()[AnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return true
		}
		return lastCheckedTime.Add(MinCheckInterval).Before(time.Now())
	}
	return e.MetaOld.GetGeneration() != e.MetaNew.GetGeneration()
}

func (f EventFilter) GenericEventFilter(e event.GenericEvent) bool {
	if lastChecked, ok := e.Meta.GetAnnotations()[AnnotationCheckedTime]; ok {
		lastCheckedTime, err := time.Parse(time.RFC3339, lastChecked)
		if err != nil {
			return true
		}
		return lastCheckedTime.Add(MinCheckInterval).Before(time.Now())
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
