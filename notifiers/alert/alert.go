package alert

import (
	"bytes"
	"text/template"
)

type Status string

const (
	MessageTemplateVariableSeverity     = "{{.Severity}}"
	MessageTemplateVariableResourceKind = "{{.ResourceKind}}"
	MessageTemplateVariableResourceName = "{{.ResourceName}}"
	MessageTemplateVariableMessage      = "{{.Message}}"

	DefaultMessageTemplate = "[" + MessageTemplateVariableSeverity + "] " + MessageTemplateVariableResourceKind + " `" + MessageTemplateVariableResourceName + "` " + MessageTemplateVariableMessage

	StatusPending    Status = "pending"    // pending to send alert
	StatusFiring     Status = "firing"     // alert currently firing
	StatusRecovering Status = "recovering" // alert is recovering, but not yet notified to external systems
	StatusError      Status = "error"

	ColorGray   = "#B2B2B2"
	ColorRed    = "#FF1717"
	ColorOrange = "#FF7400"
	ColorYellow = "#FFF400"
	ColorBlue   = "#0092FF"
	ColorGreen  = "#49FF00"
)

type Alert struct {
	// Suppressed means if this notification has been suppressed, can be used to temporary reduce the noise
	Suppressed bool `json:"suppressed"`
	// Severity is the alert severity
	Severity Severity `json:"severity"`
	// MessageTemplate is the message template for the alert
	MessageTemplate string `json:"-"`
	// Message is the message for the violation
	Message string `json:"message"`
	// ResourceKind is the resource's kind that has issue, e.g., hpa, pdb, pod, service, etc.
	ResourceKind string `json:"resourceKind"`
	// ResourceName is the resource's name, with namespace, same as types.NamespacedName.String()
	ResourceName string `json:"resourceName"`
	// Status is the status of this rule, can be pending, firing, or recovered
	Status Status `json:"status"`
	// Error is the err from any issues for sending message to external system
	Error string `json:"error"`
}

func (a Alert) ParseMessage() (string, error) {
	messageTemplate := a.MessageTemplate
	if messageTemplate == "" {
		messageTemplate = DefaultMessageTemplate
	}
	messageVariables := MessageTemplateVariables{
		Severity:     a.Severity,
		ResourceKind: a.ResourceKind,
		ResourceName: a.ResourceName,
		Message:      a.Message,
	}
	t, err := template.New("msg").Parse(messageTemplate)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := t.Execute(&buf, messageVariables); err != nil {
		return "", err
	}
	return buf.String(), nil
}

type MessageTemplateVariables struct {
	Severity     Severity
	ResourceKind string
	ResourceName string
	Message      string
}

// Severity indicates the severity of the alert
type Severity string

const (
	SeverityDefault  Severity = ""
	SeverityFatal    Severity = "fatal"
	SeverityCritical Severity = "critical"
	SeverityWarning  Severity = "warning"
	SeverityInfo     Severity = "info"
)

func (s Severity) Color() string {
	switch s {
	case SeverityFatal:
		return ColorRed
	case SeverityCritical:
		return ColorOrange
	case SeverityWarning:
		return ColorYellow
	case SeverityInfo:
		return ColorBlue
	default:
		// also for SeverityDefault
		return ColorGray
	}
}
