package notification

import (
	"fmt"

	"github.com/kasuha07/subdux/internal/service/serviceerr"
)

var (
	ErrChannelNotFound               = serviceerr.New(serviceerr.KindNotFound, "channel not found")
	ErrInvalidChannelType            = serviceerr.New(serviceerr.KindInvalid, "invalid channel type, must be one of: smtp, resend, telegram, webhook, gotify, ntfy, bark, serverchan, feishu, wecom, dingtalk, pushdeer, pushplus, pushover, napcat")
	ErrTemplateNotFound              = serviceerr.New(serviceerr.KindNotFound, "template not found")
	ErrNoTemplateForChannel          = serviceerr.New(serviceerr.KindNotFound, "no template configured (default or channel-specific)")
	ErrInvalidChannelTypeForTemplate = serviceerr.New(serviceerr.KindInvalid, "invalid channel type")
	ErrDefaultTemplateExists         = serviceerr.New(serviceerr.KindConflict, "default template already exists")
	ErrChannelTemplateExists         = serviceerr.New(serviceerr.KindConflict, "template for this channel type already exists")
)

func errTooManyEnabledChannels() *serviceerr.Error {
	return serviceerr.New(serviceerr.KindInvalid, fmt.Sprintf("you can enable at most %d notification channels", maxEnabledNotificationChannels))
}

func notificationUserFixableError(err error) error {
	if err == nil {
		return nil
	}
	return serviceerr.Wrap(serviceerr.KindInvalid, err.Error(), err)
}

func notificationUserFixableMessage(msg string, cause error) error {
	return serviceerr.Wrap(serviceerr.KindInvalid, msg, cause)
}
