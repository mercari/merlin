package slack

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/kouzoh/merlin/notifiers/alert"
)

type BlockType string
type TextType string

const (
	BlockTypeSection BlockType = "section"

	TextTypeMarkdown  TextType = "mrkdwn"
	TextTypePlainText TextType = "plain_text"
)

type Slack struct {
	// Severity is the severity of the issue, one of info, warning, critical, or fatal
	Severity alert.Severity `json:"severity"`
	// WebhookURL is the WebhookURL from slack
	WebhookURL string `json:"webhookURL"`
	// Channel is the slack channel this notification should use
	Channel string `json:"channel"`
}

type Request struct {
	Channel     string       `json:"channel"`
	IconEmoji   string       `json:"icon_emoji"`
	Username    string       `json:"username"`
	Attachments []Attachment `json:"attachments,omitempty"`
}

// see https://api.slack.com/reference/block-kit/blocks for reference
type Block struct {
	Type   BlockType           `json:"type"`
	Text   *BlockSectionText   `json:"text,omitempty"`
	Fields []BlockSectionField `json:"fields,omitempty"`
}

type Attachment struct {
	Color  string  `json:"color"`
	Blocks []Block `json:"blocks"`
}

type BlockSectionText struct {
	Type TextType `json:"type,omitempty"`
	Text string   `json:"text,omitempty"`
}

type BlockSectionField struct {
	Type  TextType `json:"type"`
	Text  string   `json:"text"`
	Emoji bool     `json:"emoji,omitempty"`
}

func (s *Slack) SendAlert(client *http.Client, a alert.Alert) error {
	if a.ResourceKind == "" || a.ResourceName == "" || a.Message == "" {
		return fmt.Errorf("alert's ResourceKind, ResourceName, and Message are required")
	}
	if a.Severity == alert.SeverityDefault {
		a.Severity = s.Severity
	}
	messagePrefix := "*[Alerting]* "
	color := a.Severity.Color()
	if a.Status == alert.StatusRecovering {
		messagePrefix = "*[Recovered]* "
		color = alert.ColorGreen
	}

	message, err := a.ParseMessage()
	if err != nil {
		return err
	}
	slackBody, _ := json.Marshal(Request{
		Channel:   s.Channel,
		IconEmoji: ":merlin:",
		Username:  "Merlin",
		Attachments: []Attachment{
			{
				Color: color,
				Blocks: []Block{
					{
						Type: BlockTypeSection,
						Text: &BlockSectionText{
							Type: TextTypeMarkdown,
							Text: messagePrefix + message,
						},
					},
				},
			},
		},
	})
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if string(body) != "ok" {
		return fmt.Errorf("non-ok response returned from slack: %s", string(body))
	}
	return nil
}
