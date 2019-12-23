package notifiers

type Slack struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}
