package v1

import (
	"github.com/kouzoh/merlin/alert"
)

// RequiredLabel is the
type RequiredLabel struct {
	// Key is the label key name
	Key string `json:"key"`
	// Value is the label value, when match is set as "regexp", the acceptable syntax of regex is RE2 (https://github.com/google/re2/wiki/Syntax)
	Value string `json:"value"`
	// Match is the way of matching, default to "exact" match, can also use "regexp" and set value to a regular express for matching.
	Match string `json:"match,omitempty"`
}

// Selector is the resource selector that used when listing kubernetes objects, only namespaced rules have this since cluster rules apply for all objects.
type Selector struct {
	// Name is the resource name this selector will select
	Name string `json:"name,omitempty"`
	// MatchLabels is the map of labels this selector will select on
	MatchLabels map[string]string `json:"matchLabels,omitempty"`
}

type Notification struct {
	// Notifiers is the list of notifiers for this notification to send
	Notifiers []string `json:"notifiers"`
	// Suppressed means if this notification has been suppressed, used for temporary reduced the noise
	Suppressed bool `json:"suppressed,omitempty"`
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity alert.Severity `json:"severity,omitempty"`
	// CustomMessageTemplate can used for customized message, variables can be used are "ResourceName, Severity, and Message"
	CustomMessageTemplate string `json:"customMessageTemplate,omitempty"`
}
