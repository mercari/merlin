package slack

import (
	"encoding/json"
	"github.com/kouzoh/merlin/notifiers/alert"
	"github.com/stretchr/testify/assert"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestSlack_SendAlert(t *testing.T) {

	channel := "test-channel"
	cases := []struct {
		desc    string
		a       alert.Alert
		want    []Attachment
		wantErr bool
	}{
		{
			desc:    "Empty alert should get err",
			a:       alert.Alert{},
			wantErr: true,
		},
		{
			desc: "regular alert should get proper attachments",
			a: alert.Alert{
				Severity:         alert.SeverityWarning,
				ResourceKind:     "Kind",
				ResourceName:     "NS/Name",
				ViolationMessage: "msg",
			},
			want: []Attachment{
				{
					Color: alert.ColorYellow,
					Blocks: []Block{
						{
							Type: BlockTypeSection,
							Text: &BlockSectionText{
								Type: TextTypeMarkdown,
								Text: "*[Alerting]* [warning] Kind `NS/Name` msg",
							},
						},
						{
							Type: BlockTypeSection,
							Fields: []BlockSectionField{
								{Type: TextTypeMarkdown, Text: "*Severity*"}, {Type: TextTypePlainText, Text: string(alert.SeverityWarning)},
								{Type: TextTypeMarkdown, Text: "*Kind*"}, {Type: TextTypePlainText, Text: "Kind"},
								{Type: TextTypeMarkdown, Text: "*Name*"}, {Type: TextTypePlainText, Text: "NS/Name"},
							},
						},
					},
				},
			},
		},
		{
			desc: "Default severity alert should get info",
			a: alert.Alert{
				ResourceKind:     "Kind",
				ResourceName:     "NS/Name",
				ViolationMessage: "msg",
			},
			want: []Attachment{
				{
					Color: alert.ColorBlue,
					Blocks: []Block{
						{
							Type: BlockTypeSection,
							Text: &BlockSectionText{
								Type: TextTypeMarkdown,
								Text: "*[Alerting]* [info] Kind `NS/Name` msg",
							},
						},
						{
							Type: BlockTypeSection,
							Fields: []BlockSectionField{
								{Type: TextTypeMarkdown, Text: "*Severity*"}, {Type: TextTypePlainText, Text: "info"},
								{Type: TextTypeMarkdown, Text: "*Kind*"}, {Type: TextTypePlainText, Text: "Kind"},
								{Type: TextTypeMarkdown, Text: "*Name*"}, {Type: TextTypePlainText, Text: "NS/Name"},
							},
						},
					},
				},
			},
		},
		{
			desc: "Recovered alert should get green color",
			a: alert.Alert{
				Status:           alert.StatusRecovering,
				ResourceKind:     "Kind",
				ResourceName:     "NS/Name",
				ViolationMessage: "msg",
			},
			want: []Attachment{
				{
					Color: alert.ColorGreen,
					Blocks: []Block{
						{
							Type: BlockTypeSection,
							Text: &BlockSectionText{
								Type: TextTypeMarkdown,
								Text: "*[Recovered]* [info] Kind `NS/Name` msg",
							},
						},
						{
							Type: BlockTypeSection,
							Fields: []BlockSectionField{
								{Type: TextTypeMarkdown, Text: "*Severity*"}, {Type: TextTypePlainText, Text: "info"},
								{Type: TextTypeMarkdown, Text: "*Kind*"}, {Type: TextTypePlainText, Text: "Kind"},
								{Type: TextTypeMarkdown, Text: "*Name*"}, {Type: TextTypePlainText, Text: "NS/Name"},
							},
						},
					},
				},
			},
		},
	}

	client := &http.Client{Timeout: 10 * time.Second}
	for _, tc := range cases {
		t.Run(tc.desc, func(tt *testing.T) {
			m := http.NewServeMux()
			m.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
				req := &Request{}
				body, err := ioutil.ReadAll(r.Body)
				assert.NoError(t, err)
				assert.NoError(t, json.Unmarshal(body, req))
				assert.Equal(t, channel, req.Channel)
				assert.Equal(t, "Merlin", req.Username)
				assert.Equal(t, ":merlin:", req.IconEmoji)
				assert.Equal(t, tc.want, req.Attachments)
				w.WriteHeader(200)
				w.Write([]byte(`ok`))
			})

			ts := httptest.NewServer(m)
			defer ts.Close()
			s := Slack{
				Severity:   alert.SeverityInfo,
				WebhookURL: ts.URL,
				Channel:    channel,
			}
			if tc.wantErr {
				assert.Error(t, s.SendAlert(client, tc.a))
			} else {
				assert.NoError(t, s.SendAlert(client, tc.a))

			}
		})
	}

}
