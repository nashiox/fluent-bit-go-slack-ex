package main

import (
	"C"
	"fmt"
	"log"
	"strconv"
	"strings"
	"unsafe"

	"github.com/fluent/fluent-bit-go/output"
	jsoniter "github.com/json-iterator/go"
)

var (
	slackClient SlackClient
	err         error
	version     string
	revision    string
)

func init() {
	log.SetFlags(log.Lshortfile)
}

//export FLBPluginRegister
func FLBPluginRegister(ctx unsafe.Pointer) int {
	return output.FLBPluginRegister(ctx, "slack_ex", "Slack Extra output plugin written in GO!")
}

//export FLBPluginInit
func FLBPluginInit(plugin unsafe.Pointer) int {
	log.Printf("[info] out_slack_ex version: %s, revision: %s initializing...", version, revision)

	ctx := map[string]interface{}{}
	ctx["username"] = output.FLBPluginConfigKey(plugin, "UserName")

	if linkNames := output.FLBPluginConfigKey(plugin, "LinkNames"); linkNames != "" {
		ctx["link_names"], err = strconv.ParseBool(linkNames)
		if err != nil {
			output.FLBPluginUnregister(plugin)
			log.Fatal(err)
			return output.FLB_ERROR
		}
	} else {
		ctx["link_names"] = false
	}

	ctx["channel"] = output.FLBPluginConfigKey(plugin, "Channel")
	if ctx["channel"] != "" && !strings.HasPrefix(ctx["channel"].(string), "#") && !strings.HasPrefix(ctx["channel"].(string), "@") {
		ctx["channel"] = "#" + ctx["channel"].(string)
	}

	if channelKeys := output.FLBPluginConfigKey(plugin, "ChannelKeys"); channelKeys != "" {
		ctx["channel_keys"] = strings.Split(channelKeys, ",")
	} else {
		ctx["channel_keys"] = []string{}
	}

	if ctx["channel"] != "" && len(ctx["channel_keys"].([]string)) > 0 {
		if strings.Contains(fmt.Sprintf(ctx["channel"].(string), ctx["channel_keys"]), "%!") {
			output.FLBPluginUnregister(plugin)
			log.Fatalf("string specifier '%%s' for `channel` and `channel_keys` specification mismatch\n")
			return output.FLB_ERROR
		}
	}

	webhookURL := output.FLBPluginConfigKey(plugin, "WebhookURL")
	if webhookURL == "" {
		output.FLBPluginUnregister(plugin)
		log.Fatal("`webhook_url` is required")
		return output.FLB_ERROR
	}

	slackClient = NewIncommingWebhook(webhookURL)

	ctx["message"] = output.FLBPluginConfigKey(plugin, "Message")
	if ctx["message"] == "" {
		ctx["message"] = "%v"
	}

	if messageKeys := output.FLBPluginConfigKey(plugin, "MessageKeys"); messageKeys != "" {
		ctx["message_keys"] = strings.Split(messageKeys, ",")
	} else {
		ctx["message_keys"] = []string{"message"}
	}

	if strings.Contains(fmt.Sprintf(ctx["message"].(string), ctx["message_keys"]), "%!") {
		output.FLBPluginUnregister(plugin)
		log.Fatalf("string specifier '%%s' for `message` and `message_keys` specification mismatch\n")
		return output.FLB_ERROR
	}

	ctx["title"] = output.FLBPluginConfigKey(plugin, "Title")

	if titleKeys := output.FLBPluginConfigKey(plugin, "TitleKeys"); titleKeys != "" {
		ctx["title_keys"] = strings.Split(titleKeys, ",")
	} else {
		ctx["title_keys"] = []string{}
	}

	if ctx["title"] != "" && len(ctx["title_keys"].([]string)) > 0 {
		if strings.Contains(fmt.Sprintf(ctx["title"].(string), ctx["title_keys"]), "%!") {
			output.FLBPluginUnregister(plugin)
			log.Fatalf("string specifier '%%s' for `title` and `title_keys` specification mismatch\n")
			return output.FLB_ERROR
		}
	}

	ctx["icon_emoji"] = output.FLBPluginConfigKey(plugin, "IconEmoji")
	ctx["icon_url"] = output.FLBPluginConfigKey(plugin, "IconURL")
	if ctx["icon_emoji"] == "" && ctx["icon_url"] == "" {
		output.FLBPluginUnregister(plugin)
		log.Fatal("either of `icon_emoji` or `icon_url` can be specified")
		return output.FLB_ERROR
	}

	if mrkdwn := output.FLBPluginConfigKey(plugin, "Mrkdwn"); mrkdwn != "" {
		ctx["mrkdwn"], err = strconv.ParseBool(mrkdwn)
		if err != nil {
			output.FLBPluginUnregister(plugin)
			log.Fatal(err)
			return output.FLB_ERROR
		}
	} else {
		ctx["mrkdwn"] = false
	}

	ctx["mrkdwn_in"] = []string{}
	if ctx["mrkdwn"].(bool) {
		ctx["mrkdwn_in"] = []string{"text", "fields"}
	}

	ctx["parse"] = output.FLBPluginConfigKey(plugin, "Parse")
	if ctx["parse"] == "none" || ctx["parse"] == "full" {
		output.FLBPluginUnregister(plugin)
		log.Fatal("`parse` must be either of `none` or `full`")
		return output.FLB_ERROR
	}

	ctx["color"] = output.FLBPluginConfigKey(plugin, "Color")

	if verboseFallback := output.FLBPluginConfigKey(plugin, "VerboseFallback"); verboseFallback != "" {
		ctx["verbose_fallback"], err = strconv.ParseBool(verboseFallback)
		if err != nil {
			output.FLBPluginUnregister(plugin)
			log.Fatal(err)
			return output.FLB_ERROR
		}
	} else {
		ctx["verbose_fallback"] = false
	}

	output.FLBPluginSetContext(plugin, ctx)

	return output.FLB_OK
}

//export FLBPluginFlushCtx
func FLBPluginFlushCtx(ctx, data unsafe.Pointer, length C.int, tag *C.char) int {
	c := output.FLBPluginGetContext(ctx).(map[string]interface{})

	log.Printf("[event] Flush called, context %s, %s\n", c["username"], C.GoString(tag))
	dec := output.NewDecoder(data, int(length))

	for {
		ret, _, record := output.GetRecord(dec)
		if ret != 0 {
			break
		}

		line, err := createJSON(C.GoString(tag), record, c)
		if err != nil {
			log.Printf("[warning] error creating message for Slack: %v\n", err)
			continue
		}

		if err := slackClient.PostMessage(line); err != nil {
			log.Printf("[warning] error sending message in Slack: %v\n", err)
			return output.FLB_RETRY
		}
	}

	return output.FLB_OK
}

func createJSON(tag string, record map[interface{}]interface{}, ctx map[string]interface{}) ([]byte, error) {
	var payload *SlackPayload
	var err error
	if ctx["title"] != "" {
		payload, err = createTitlePayload(tag, record, ctx)
	} else if ctx["color"] != "" {
		payload, err = createColorPayload(tag, record, ctx)
	} else {
		payload, err = createPlainPayload(tag, record, ctx)
	}
	if err != nil {
		return []byte("{}"), err
	}

	if ctx["username"] != "" {
		payload.UserName = ctx["username"].(string)
	}

	if ctx["icon_emoji"] != "" {
		payload.IconEmoji = ctx["icon_emoji"].(string)
	}

	if ctx["icon_url"] != "" {
		payload.IconURL = ctx["icon_url"].(string)
	}

	if ctx["mrkdwn"].(bool) {
		payload.Mrkdwn = ctx["mrkdwn"].(bool)
	}

	if ctx["link_names"].(bool) {
		payload.Mrkdwn = ctx["link_names"].(bool)
	}

	if ctx["parse"] != "" {
		payload.IconURL = ctx["parse"].(string)
	}

	if len(payload.Attachments) > 0 {
		for i := range payload.Attachments {
			if ctx["color"] != "" {
				payload.Attachments[i].Color = ctx["color"].(string)
			}

			if len(ctx["mrkdwn_in"].([]string)) > 0 {
				payload.Attachments[i].MrkdwnIn = ctx["mrkdwn_in"].([]string)
			}
		}
	} else {
		attach := SlackAttachment{}
		if ctx["color"] != "" {
			attach.Color = ctx["color"].(string)
		}

		if len(ctx["mrkdwn_in"].([]string)) > 0 {
			attach.MrkdwnIn = ctx["mrkdwn_in"].([]string)
		}

		payload.Attachments = append(payload.Attachments, attach)
	}

	json := jsoniter.ConfigCompatibleWithStandardLibrary
	js, err := json.Marshal(payload)
	if err != nil {
		return []byte("{}"), err
	}

	return js, nil
}

func createTitlePayload(tag string, record map[interface{}]interface{}, ctx map[string]interface{}) (*SlackPayload, error) {
	payload := SlackPayload{}
	r := parseMap(record)

	channel, err := buildChannel(r, ctx)
	if err != nil {
		return nil, err
	}

	if channel != "" {
		payload.Channel = channel
	}

	title, err := buildTitle(r, ctx)
	if err != nil {
		return nil, err
	}

	message, err := buildMessage(r, ctx)
	if err != nil {
		return nil, err
	}

	fallbackText := title
	if ctx["verbose_fallback"].(bool) {
		fallbackText = fmt.Sprintf("%s %s", title, message)
	}

	payload.Attachments = []SlackAttachment{
		{
			Fallback: fallbackText,
			Fields: []SlackField{
				{
					Title: title,
					Value: message,
				},
			},
		},
	}

	return &payload, nil
}

func createColorPayload(tag string, record map[interface{}]interface{}, ctx map[string]interface{}) (*SlackPayload, error) {
	payload := SlackPayload{}
	r := parseMap(record)

	channel, err := buildChannel(r, ctx)
	if err != nil {
		return nil, err
	}

	if channel != "" {
		payload.Channel = channel
	}

	message, err := buildMessage(r, ctx)
	if err != nil {
		return nil, err
	}

	payload.Attachments = []SlackAttachment{
		{
			Fallback: message,
			Text:     message,
		},
	}

	return &payload, nil
}

func createPlainPayload(tag string, record map[interface{}]interface{}, ctx map[string]interface{}) (*SlackPayload, error) {
	payload := SlackPayload{}
	r := parseMap(record)

	channel, err := buildChannel(r, ctx)
	if err != nil {
		return nil, err
	}

	if channel != "" {
		payload.Channel = channel
	}

	message, err := buildMessage(r, ctx)
	if err != nil {
		return nil, err
	}

	payload.Text = message
	payload.Attachments = []SlackAttachment{}

	return &payload, nil
}

func parseMap(mapInterface map[interface{}]interface{}) map[string]interface{} {
	m := make(map[string]interface{})

	for k, v := range mapInterface {
		switch t := v.(type) {
		case []byte:
			m[k.(string)] = string(t)
		case map[interface{}]interface{}:
			m[k.(string)] = parseMap(t)
		default:
			m[k.(string)] = v
		}
	}

	return m
}

func buildChannel(record map[string]interface{}, ctx map[string]interface{}) (string, error) {
	if ctx["channel"] == "" {
		return "", nil
	}

	if len(ctx["channel_keys"].([]string)) == 0 {
		return ctx["channel"].(string), nil
	}

	values, err := fetchKeys(record, ctx["channel_keys"].([]string))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(ctx["channel"].(string), values...), nil
}

func buildTitle(record map[string]interface{}, ctx map[string]interface{}) (string, error) {
	if len(ctx["title_keys"].([]string)) == 0 {
		return ctx["title"].(string), nil
	}

	values, err := fetchKeys(record, ctx["title_keys"].([]string))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(ctx["title"].(string), values...), nil
}

func buildMessage(record map[string]interface{}, ctx map[string]interface{}) (string, error) {
	values, err := fetchKeys(record, ctx["message_keys"].([]string))
	if err != nil {
		return "", err
	}

	return fmt.Sprintf(ctx["message"].(string), values...), nil
}

func fetchKeys(record map[string]interface{}, keys []string) ([]interface{}, error) {
	values := make([]interface{}, len(keys), len(keys))
	for k, v := range keys {
		if _, ok := record[v]; !ok {
			return nil, fmt.Errorf("out_slack_ex: the specified key '%s' not found in record. [%v]", v, record)
		}
		values[k] = record[v]
	}

	return values, nil
}

//export FLBPluginExit
func FLBPluginExit() int {
	return output.FLB_OK
}

func main() {}
