package controllers

import (
	"fmt"
	"reflect"
	"sync"

	merlinv1 "github.com/kouzoh/merlin/api/v1"
)

const (
	metadataNameField = ".metadata.name"
)

var NotifierNotFoundErr = fmt.Errorf("notifier not found")

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

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}
