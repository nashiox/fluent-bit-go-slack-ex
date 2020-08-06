package main

import (
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
)

type SlackClient interface {
	PostMessage(payload []byte) error
}

type SlackWebbookClient struct {
	webhookURL string
}

type SlackPayload struct {
	UserName  string `json:"username,omitempty"`
	IconEmoji string `json:"icon_emoji,omitempty"`
	IconURL   string `json:"icon_url,omitempty"`
	Mrkdwn    bool   `json:"mrkdwn,omitempty"`
	LinkNames bool   `json:"link_names,omitempty"`
	Parse     string `json:"parse,omitempty"`
	Channel   string `json:"channel,omitempty"`
	Text      string `json:"text,omitempty"`

	Attachments []SlackAttachment `json:"attachments,omitempty"`
}

type SlackAttachment struct {
	Fallback string   `json:"fallback,omitempty"`
	Text     string   `json:"text,omitempty"`
	PreText  string   `json:"pretext,omitempty"`
	Color    string   `json:"color,omitempty"`
	MrkdwnIn []string `json:"mrkdwn_in,omitempty"`

	Fields []SlackField `json:"fields,omitempty"`
}

type SlackField struct {
	Title string `json:"title,omitempty"`
	Value string `json:"value,omitempty"`
}

func NewIncommingWebhook(webhookURL string) *SlackWebbookClient {
	return &SlackWebbookClient{webhookURL: webhookURL}
}

func (c *SlackWebbookClient) PostMessage(payload []byte) error {
	resp, err := http.PostForm(c.webhookURL, url.Values{"payload": {string(payload)}})
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || 300 <= resp.StatusCode {
		body, err := ioutil.ReadAll(resp.Body)
		if err != nil {
			return err
		}

		return fmt.Errorf("slack web API replied %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
