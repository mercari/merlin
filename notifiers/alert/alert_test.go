package alert

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAlert_ParseMessage(t *testing.T) {
	a := Alert{
		Severity:     SeverityInfo,
		Message:      "test violation",
		ResourceKind: "hpa",
		ResourceName: "default/test-hpa",
		Status:       StatusPending,
	}

	msg, err := a.ParseMessage()
	assert.NoError(t, err)
	assert.Equal(t, "[info] hpa `default/test-hpa` test violation", msg)
}

func TestSeverity_Color(t *testing.T) {
	cases := []struct {
		desc     string
		severity Severity
		color    string
	}{
		{desc: "SeverityFatal Color should be Red", severity: SeverityFatal, color: ColorRed},
		{desc: "SeverityCritical Color should be Orange", severity: SeverityCritical, color: ColorOrange},
		{desc: "SeverityWarning Color should be Yellow", severity: SeverityWarning, color: ColorYellow},
		{desc: "SeverityInfo Color should be Blue", severity: SeverityInfo, color: ColorBlue},
		{desc: "SeverityDefault Color should be Gray", severity: SeverityDefault, color: ColorGray},
	}

	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			assert.Equal(t, tc.color, tc.severity.Color())
		})
	}
}
