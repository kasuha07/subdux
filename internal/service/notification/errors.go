package notification

import (
	"fmt"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

var (
	ErrChannelNotFound               = serviceerr.New(serviceerr.KindNotFound, "channel_not_found", "channel not found")
	ErrInvalidChannelType            = serviceerr.New(serviceerr.KindInvalid, "invalid_channel_type_must_be_one_of_smtp_resend_telegram_webhook_gotify_ntfy_bark_serverchan_feishu_wecom_dingtalk_pushdeer_pushplus_pushover_napcat", "invalid channel type, must be one of: smtp, resend, telegram, webhook, gotify, ntfy, bark, serverchan, feishu, wecom, dingtalk, pushdeer, pushplus, pushover, napcat")
	ErrTemplateNotFound              = serviceerr.New(serviceerr.KindNotFound, "template_not_found", "template not found")
	ErrNoTemplateForChannel          = serviceerr.New(serviceerr.KindNotFound, "no_template_configured_default_or_channel_specific", "no template configured (default or channel-specific)")
	ErrInvalidChannelTypeForTemplate = serviceerr.New(serviceerr.KindInvalid, "invalid_channel_type", "invalid channel type")
	ErrDefaultTemplateExists         = serviceerr.New(serviceerr.KindConflict, "default_template_already_exists", "default template already exists")
	ErrChannelTemplateExists         = serviceerr.New(serviceerr.KindConflict, "template_for_this_channel_type_already_exists", "template for this channel type already exists")
)

func errTooManyEnabledChannels() *serviceerr.Error {
	return serviceerr.NewCode(
		serviceerr.KindInvalid,
		"maximum_number_of_notification_channels_reached",
		fmt.Sprintf("you can enable at most %d notification channels", maxEnabledNotificationChannels),
		map[string]any{"max": maxEnabledNotificationChannels},
	)
}

func notificationUserFixableError(err error) error {
	if err == nil {
		return nil
	}
	return serviceerr.Wrap(serviceerr.KindInvalid, "notification_action_failed", err.Error(), err)
}

func notificationUserFixableMessage(msg string, cause error) error {
	return serviceerr.Wrap(serviceerr.KindInvalid, "notification_action_failed", msg, cause)
}
