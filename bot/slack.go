package bot

import (
	"context"

	"github.com/slack-go/slack"
)

// SlackClient - a partial interface to slack.Client
type SlackClient interface {
	SendMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (_channel, _timestamp, _text string, err error)
	GetUserGroupMembersContext(ctx context.Context, userGroup string) ([]string, error)
	OpenViewContext(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error)
	UpdateViewContext(ctx context.Context, view slack.ModalViewRequest, externalID, hash, viewID string) (*slack.ViewResponse, error)
	CreateConversationContext(ctx context.Context, channelName string, isPrivate bool) (*slack.Channel, error)
	GetConversationInfoContext(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error)
	ArchiveConversationContext(ctx context.Context, channelID string) error
	SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error)
	SetTopicOfConversationContext(ctx context.Context, channelID, topic string) (*slack.Channel, error)
	InviteUsersToConversationContext(ctx context.Context, channelID string, users ...string) (*slack.Channel, error)
	AddChannelReminder(channelID string, text string, time string) (*slack.Reminder, error)
	GetUserInfoContext(ctx context.Context, user string) (*slack.User, error)
	AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error)
	GetConversationsForUserContext(ctx context.Context, params *slack.GetConversationsForUserParameters) ([]slack.Channel, string, error)
}
