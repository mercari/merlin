package notifiers

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"time"
)

type Slack struct {
	WebhookURL string `json:"webhook_url"`
	Channel    string `json:"channel"`
}

type SlackRequest struct {
	Text      string `json:"text"`
	Channel   string `json:"channel"`
	IconEmoji string `json:"icon_emoji"`
	Username  string `json:"username"`
}

func (s *Slack) SendMessage(msg string) error {
	slackBody, _ := json.Marshal(SlackRequest{Text: msg, Channel: s.Channel, IconEmoji: ":merlin:", Username: "Merlin"})
	req, err := http.NewRequest(http.MethodPost, s.WebhookURL, bytes.NewBuffer(slackBody))
	if err != nil {
		return err
	}

	req.Header.Add("Content-Type", "application/json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	buf := new(bytes.Buffer)
	_, err = buf.ReadFrom(resp.Body)
	if err != nil {
		return err
	}
	if buf.String() != "ok" {
		return errors.New("non-ok response returned from Slack")
	}
	return nil
}
