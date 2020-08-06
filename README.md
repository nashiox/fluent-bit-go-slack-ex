# fluent-bit slack extra output plugin

This plugin works with fluent-bit's go plugin interface.
You can send a message to slack using fluent-bit-go-slack-ex inspired by [fluent-plugin-slack](https://github.com/sowawa/fluent-plugin-slack).

The configuration typically looks like:

```
Fluent-Bit --> Slack
```

## Usage

```bash
$ fluent-bit -e /path/to/built/out_slack_ex.so -c fluent-bit.conf
```

## Prerequisites

- Go 1.14+
- gcc (for cgo)

## Building

Library:

```bash
$ make build
```

Container Image:


```bash
$ docker build -t fluent-bit-go-slack-ex .
```

### Configuration Options

|  Key  |  Description  |  Default value  |
| ---- | ---- | ---- |
|  WebhookURL  |  Incoming Webhook URL. See [https://api.slack.com/incoming-webhooks](https://api.slack.com/incoming-webhooks) (Required)  |  -  |
|  UserName  |  name of bot  |  -  |
|  LinkNames  |  find and link channel names and usernames.NOTE: This parameter must be `true` to receive Desktop Notification via Mentions in cases of Incoming Webhook and Slack Web API  |  false  |
|  Channel  |  Channel name or id to send messages (without first '#'). Channel ID is recommended because it is unchanged even if a channel is renamed  |  -  |
|  ChannelKeys  |  keys used to format channel. `%s` will be replaced with value specified by channel_keys if this option is used  |  -  |
|  Title  |  title format. `%s` will be replaced with value specified by title_keys. title is created from the first appeared record on each tag. NOTE: This parameter must not be specified to receive Desktop Notification via Mentions in cases of Incoming Webhook and Slack Web API  |  -  |
|  TitleKeys  |  keys used to format the title  |  -  |
|  Message  |  message format. `%s` will be replaced with value specified by message_keys  |  `%s`  |
|  MessageKeys  |  keys used to format messages  |  message  |
|  IconEmoji  |  emoji to use as the icon. either of `IconEmoji` or `IconURL` can be specified  |  -  |
|  IconURL  |  url to an image to use as the icon. either of `IconEmoji` or `IconURL` can be specified  |  -  |
|  Mrkdwn  |  enable formatting. see [https://api.slack.com/docs/formatting](https://api.slack.com/docs/formatting)  |  false  |
|  Parse  |  change how messages are treated. none or full can be specified. See Parsing mode section of [https://api.slack.com/docs/formatting](https://api.slack.com/docs/formatting)  |  -  |
|  Color  |  color to use such as `good` or `bad` . See Color section of [https://api.slack.com/docs/attachments](https://api.slack.com/docs/attachments). NOTE: This parameter must not be specified to receive Desktop Notification via Mentions in cases of Incoming Webhook and Slack Web API  |  -  |
|  VerboseFallback  |  Originally, only `title` is used for the fallback which is the message shown on popup if `title` is given. If this option is set to be `true` , messages are also included to the fallback attribute  |  false  |

Example:

add this section to fluent-bit.conf

```conf
[OUTPUT]
    Name slack_ex
    Match *
    WebhookURL ${FLUENT_BIT_WEBHOOKURL}
    Color #FFFF00
    Mrkdwn false
    Channel %s
    ChannelKeys channel
    UserName msg from fluentd
    Title %s
    TitleKeys from
    IconEmoji :information_source:
```
