package controllers

import (
	"reflect"
	"sync"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

const (
	AnnotationCheckedTime = merlinv1.GROUP + "/checked-at"
	AnnotationIssue       = merlinv1.GROUP + "/issue"
	indexField            = ".metadata.name"
)

func GetStructName(v interface{}) string {
	if t := reflect.TypeOf(v); t.Kind() == reflect.Ptr {
		return t.Elem().Name()
	} else {
		return t.Name()
	}
}

// RuleStatusWithLock is the status of rule with lock
type RuleStatusWithLock struct {
	sync.Mutex
	merlinv1.RuleStatus
}
