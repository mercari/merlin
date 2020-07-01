package controllers

import (
	"fmt"
	"math/rand"
	"reflect"
	"time"
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

// Helper functions to check and remove string from a slice of strings.
func containsString(slice []string, s string) bool {
	for _, item := range slice {
		if item == s {
			return true
		}
	}
	return false
}

func requeueIntervalForError() time.Duration {
	rand.Seed(time.Now().UnixNano())
	min := 10
	max := 30
	return time.Duration(rand.Intn(max-min+1)+min) * time.Second
}
