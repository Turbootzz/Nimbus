package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/nimbus/backend/internal/models"
)

// Notification targets: ntfy, Gotify, Telegram and Pushover. Each has its own
// validation and request builder; delivery itself is shared with the plain
// webhook formats (see WebhookNotifier.deliver).

const (
	ntfyDefaultPriority = 3
	ntfyMaxPriority     = 5

	// Gotify clients add sound from 4 and vibration from 8
	gotifyPriorityOffline = 8
	gotifyPriorityOnline  = 4

	// Service limits, counted in characters
	telegramMaxText    = 4096
	pushoverMaxTitle   = 250
	pushoverMaxMessage = 1024
	pushoverMaxURL     = 512

	maxErrorDetail = 200
)

var (
	// Printable ASCII without spaces: safe in a header and in JSON
	tokenPattern = regexp.MustCompile(`^[\x21-\x7E]{1,512}$`)
	// The bot token becomes a URL path segment, so its shape is strict
	telegramTokenPattern = regexp.MustCompile(`^[0-9]{1,20}:[A-Za-z0-9_-]{1,128}$`)
	// Numeric chat id (negative for groups) or a public @channelname
	telegramChatPattern = regexp.MustCompile(`^(-?[0-9]{1,20}|@[A-Za-z0-9_]{1,64})$`)
	pushoverKeyPattern  = regexp.MustCompile(`^[A-Za-z0-9]{30}$`)
	ntfyTopicPattern    = regexp.MustCompile(`^[-_A-Za-z0-9]{1,64}$`)
)

// PrepareWebhookTarget normalises a webhook for its format (fixed URL,
// defaults, settings other formats don't use) and validates it. creds are the
// new secrets, or nil when the stored ones are kept and need no check. It
// returns field name -> message; empty means valid.
func PrepareWebhookTarget(webhook *models.Webhook, creds *models.WebhookCredentials) map[string]string {
	fields := map[string]string{}
	if fixed := models.WebhookFixedURL(webhook.Format); fixed != "" {
		webhook.URL = fixed
	}
	options := webhook.Options
	webhook.Options = models.WebhookOptions{}

	requireToken := func(pattern *regexp.Regexp, label string) {
		if creds == nil {
			return
		}
		if creds.Token == "" {
			fields["token"] = label + " is required"
		} else if !pattern.MatchString(creds.Token) {
			fields["token"] = label + " is not valid"
		}
	}

	switch webhook.Format {
	case models.WebhookFormatNtfy:
		if _, _, ok := splitNtfyTopicURL(webhook.URL); !ok && webhook.URL != "" {
			fields["url"] = "The URL must end with the topic, like https://ntfy.sh/my-topic"
		}
		webhook.Options.Priority = options.Priority
		if options.Priority == 0 {
			webhook.Options.Priority = ntfyDefaultPriority
		} else if options.Priority < 1 || options.Priority > ntfyMaxPriority {
			fields["priority"] = "Priority must be between 1 and 5"
		}
		if creds != nil {
			creds.UserKey = ""
			if creds.Token != "" && !tokenPattern.MatchString(creds.Token) {
				fields["token"] = "Access token is not valid"
			}
		}
	case models.WebhookFormatGotify:
		requireToken(tokenPattern, "App token")
		if creds != nil {
			creds.UserKey = ""
		}
	case models.WebhookFormatTelegram:
		webhook.Options.ChatID = strings.TrimSpace(options.ChatID)
		if webhook.Options.ChatID == "" {
			fields["chat_id"] = "Chat ID is required"
		} else if !telegramChatPattern.MatchString(webhook.Options.ChatID) {
			fields["chat_id"] = "Chat ID must be a number or a public @channelname"
		}
		requireToken(telegramTokenPattern, "Bot token")
		if creds != nil {
			creds.UserKey = ""
		}
	case models.WebhookFormatPushover:
		requireToken(pushoverKeyPattern, "App token")
		if creds != nil {
			if creds.UserKey == "" {
				fields["user_key"] = "User key is required"
			} else if !pushoverKeyPattern.MatchString(creds.UserKey) {
				fields["user_key"] = "User key is not valid"
			}
		}
	default:
		// Plain webhooks carry their secret in the URL and store nothing else
		if creds != nil {
			*creds = models.WebhookCredentials{}
		}
	}
	return fields
}

// splitNtfyTopicURL splits "https://ntfy.sh/alerts" into the server URL and
// the topic. JSON publishing posts to the server root with the topic in the body.
func splitNtfyTopicURL(topicURL string) (server, topic string, ok bool) {
	parsed, err := url.Parse(topicURL)
	if err != nil || parsed.Host == "" {
		return "", "", false
	}
	path := strings.TrimSuffix(parsed.Path, "/")
	cut := strings.LastIndex(path, "/")
	topic = path[cut+1:]
	if !ntfyTopicPattern.MatchString(topic) {
		return "", "", false
	}
	parsed.Path = path[:cut+1]
	parsed.RawPath, parsed.RawQuery, parsed.Fragment = "", "", ""
	return parsed.String(), topic, true
}

// notificationText is the title and body the notification targets share
func notificationText(event NotificationEvent) (title, message string) {
	if event.NewStatus == models.StatusOffline {
		message = "The service is not responding."
		if event.ErrorMsg != nil && *event.ErrorMsg != "" {
			message = *event.ErrorMsg
		}
		return event.ServiceName + " is offline", message
	}
	return event.ServiceName + " is back online", "The service is back up and responding normally."
}

// truncate cuts s to at most limit characters
func truncate(s string, limit int) string {
	runes := []rune(s)
	if len(runes) <= limit {
		return s
	}
	return string(runes[:limit-1]) + "…"
}

func buildNtfyPayload(topic string, priority int, event NotificationEvent) map[string]interface{} {
	title, message := notificationText(event)
	tag := "heavy_check_mark"
	if event.NewStatus == models.StatusOffline {
		tag = "rotating_light"
	}
	if priority == 0 {
		priority = ntfyDefaultPriority
	}
	payload := map[string]interface{}{
		"topic":    topic,
		"title":    title,
		"message":  message,
		"priority": priority,
		"tags":     []string{tag},
	}
	if event.ServiceURL != "" {
		payload["click"] = event.ServiceURL
	}
	return payload
}

func buildGotifyPayload(event NotificationEvent) map[string]interface{} {
	title, message := notificationText(event)
	priority := gotifyPriorityOnline
	if event.NewStatus == models.StatusOffline {
		priority = gotifyPriorityOffline
	}
	payload := map[string]interface{}{
		"title":    title,
		"message":  message,
		"priority": priority,
	}
	if event.ServiceURL != "" {
		payload["extras"] = map[string]interface{}{
			"client::notification": map[string]interface{}{
				"click": map[string]interface{}{"url": event.ServiceURL},
			},
		}
	}
	return payload
}

// buildTelegramPayload sends plain text (no parse_mode), so service names and
// error messages need no escaping
func buildTelegramPayload(chatID string, event NotificationEvent) map[string]interface{} {
	title, message := notificationText(event)
	icon := "🟢"
	if event.NewStatus == models.StatusOffline {
		icon = "🔴"
	}
	text := icon + " " + title + "\n" + message
	if event.ServiceURL != "" {
		text += "\n" + event.ServiceURL
	}
	return map[string]interface{}{
		"chat_id": chatID,
		"text":    truncate(text, telegramMaxText),
	}
}

func buildPushoverPayload(creds models.WebhookCredentials, event NotificationEvent) map[string]interface{} {
	title, message := notificationText(event)
	payload := map[string]interface{}{
		"token":   creds.Token,
		"user":    creds.UserKey,
		"title":   truncate(title, pushoverMaxTitle),
		"message": truncate(message, pushoverMaxMessage),
	}
	if event.ServiceURL != "" && len([]rune(event.ServiceURL)) <= pushoverMaxURL {
		payload["url"] = event.ServiceURL
		payload["url_title"] = "Open " + truncate(event.ServiceName, 90)
	}
	return payload
}

// buildRequest builds the outbound request for a webhook. Secrets only go in
// headers or the body, except Telegram's bot token: its API wants it in the
// URL path, so that URL must never be logged or returned (see deliveryError).
func (w *WebhookNotifier) buildRequest(ctx context.Context, webhook *models.Webhook, creds models.WebhookCredentials, event NotificationEvent) (*http.Request, error) {
	target := webhook.URL
	header := http.Header{}
	var payload interface{}

	switch webhook.Format {
	case models.WebhookFormatNtfy:
		server, topic, ok := splitNtfyTopicURL(webhook.URL)
		if !ok {
			return nil, errors.New("invalid ntfy topic URL")
		}
		target = server
		payload = buildNtfyPayload(topic, webhook.Options.Priority, event)
		if creds.Token != "" {
			header.Set("Authorization", "Bearer "+creds.Token)
		}
	case models.WebhookFormatGotify:
		joined, err := url.JoinPath(webhook.URL, "message")
		if err != nil {
			return nil, errors.New("invalid Gotify URL")
		}
		target = joined
		payload = buildGotifyPayload(event)
		header.Set("X-Gotify-Key", creds.Token)
	case models.WebhookFormatTelegram:
		if !telegramTokenPattern.MatchString(creds.Token) {
			return nil, errors.New("invalid Telegram bot token")
		}
		target = strings.TrimSuffix(webhook.URL, "/") + "/bot" + creds.Token + "/sendMessage"
		payload = buildTelegramPayload(webhook.Options.ChatID, event)
	case models.WebhookFormatPushover:
		payload = buildPushoverPayload(creds, event)
	default:
		payload = w.buildPayload(webhook.Format, event)
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, target, bytes.NewReader(body))
	if err != nil {
		return nil, errors.New("invalid webhook URL")
	}
	req.Header = header
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Nimbus-Webhook/1.0")
	return req, nil
}

// deliveryError turns a transport error into text that is safe to store and
// show. net/http wraps errors in *url.Error, whose message contains the full
// request URL; only the cause is kept.
func deliveryError(err error, creds models.WebhookCredentials) string {
	var urlErr *url.Error
	if errors.As(err, &urlErr) {
		err = urlErr.Err
	}
	return scrubSecrets(err.Error(), creds)
}

// responseError describes a non-2xx response, with the reason the service
// gave when it sent one (Telegram "description", Pushover "errors", Gotify
// "errorDescription", ntfy "error").
func responseError(resp *http.Response, creds models.WebhookCredentials) string {
	msg := fmt.Sprintf("HTTP %d", resp.StatusCode)
	var body struct {
		Description      string   `json:"description"`
		ErrorDescription string   `json:"errorDescription"`
		Error            string   `json:"error"`
		Errors           []string `json:"errors"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 16<<10)).Decode(&body); err != nil {
		return msg
	}
	for _, detail := range []string{body.Description, body.ErrorDescription, body.Error, strings.Join(body.Errors, ", ")} {
		if detail != "" {
			return msg + ": " + scrubSecrets(truncate(detail, maxErrorDetail), creds)
		}
	}
	return msg
}

// scrubSecrets removes credentials from a message, in case a server or
// library echoes them back
func scrubSecrets(msg string, creds models.WebhookCredentials) string {
	for _, secret := range []string{creds.Token, creds.UserKey} {
		if secret != "" {
			msg = strings.ReplaceAll(msg, secret, "[redacted]")
		}
	}
	return msg
}
