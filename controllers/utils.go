package controllers

import (
	"reflect"
	"sync"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

const (
	metadataNameField = ".metadata.name"
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
