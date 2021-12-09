package bot

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"

	"github.com/slack-go/slack"
	"github.com/stretchr/testify/assert"
)

func TestUpdateAdmins(t *testing.T) {
	ctx := context.TODO()
	c := &dummyClient{}
	b := &botHandler{
		admins:      &ugMembers{},
		slackClient: c,
	}

	err := b.updateAdmins(ctx, "randomusergroup")
	assert.NoError(t, err)
	assert.EqualValues(t, []string(nil), b.admins.members)

	c.members = []string{"user1", "user2", "user3"}

	err = b.updateAdmins(ctx, "randomusergroup")
	assert.NoError(t, err)
	assert.EqualValues(t, []string{"user1", "user2", "user3"}, b.admins.members)

	c.err = fmt.Errorf("something bad")
	err = b.updateAdmins(ctx, "randomusergroup")
	assert.Error(t, err)
}

func TestIsAdmin(t *testing.T) {
	ctx := context.TODO()
	c := &dummyClient{
		members: []string{},
		err:     fmt.Errorf(":("),
	}
	b := &botHandler{
		admins:      &ugMembers{},
		slackClient: c,
	}

	assert.False(t, b.isAdmin(ctx, "admin"))

	c.err = nil
	c.members = []string{"admin"}

	assert.False(t, b.isAdmin(ctx, "randomuser"))
	assert.True(t, b.isAdmin(ctx, "admin"))
}

func newPostRequest(body *bytes.Buffer) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/command", body)
	r.Header.Set("content-type", "application/x-www-form-urlencoded")
	return r
}

func TestHandleCommand(t *testing.T) {
	c := &dummyClient{}
	c.User = &slack.User{
		Locale: "en-US",
	}
	b := &botHandler{
		slackClient: c,
		admins: &ugMembers{
			members: []string{"adminuser"},
		},
		opts: Opts{
			AdminGroupID:       "admingroup",
			BroadcastChannelID: "channel",
		},
	}

	// invalid command body => 500
	w := httptest.NewRecorder()
	r := newPostRequest(new(bytes.Buffer))
	r.Body = nil

	b.handleCommand(w, r)
	assert.Equal(t, 500, w.Code)

	// admin user, invalid command => 404
	w = httptest.NewRecorder()
	v := url.Values{}
	v.Set("user_id", "adminuser")
	v.Set("command", "/doesnotexist")
	body := bytes.NewBufferString(v.Encode())
	r = newPostRequest(body)

	b.handleCommand(w, r)
	assert.Equal(t, 404, w.Code)

	// help arg
	w = httptest.NewRecorder()
	v = url.Values{}
	v.Set("user_id", "adminuser")
	v.Set("command", "/devopsbot")
	v.Set("text", "help")
	body = bytes.NewBufferString(v.Encode())
	r = newPostRequest(body)

	b.handleCommand(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, c.response["text"][0], "available commands:")

	// missing arg
	w = httptest.NewRecorder()
	v = url.Values{}
	v.Set("user_id", "adminuser")
	v.Set("command", "/devopsbot")
	v.Set("text", "")
	body = bytes.NewBufferString(v.Encode())
	r = newPostRequest(body)

	b.handleCommand(w, r)
	assert.Equal(t, 200, w.Code)
	assert.Contains(t, c.response["text"][0], "available commands:")
}

func TestCreateOptionBlockObjects(t *testing.T) {
	options := []string{}
	optionBlockObjects := createOptionBlockObjects(options, "")
	assert.Empty(t, optionBlockObjects)

	options = []string{"a", "b"}
	optionBlockObjects = createOptionBlockObjects(options, "")
	assert.Equal(t, []*slack.OptionBlockObject{
		(&slack.OptionBlockObject{
			Text:  &slack.TextBlockObject{Type: "plain_text", Text: "a", Emoji: false, Verbatim: false},
			Value: "a",
			URL:   ""}),
		(&slack.OptionBlockObject{
			Text:  &slack.TextBlockObject{Type: "plain_text", Text: "b", Emoji: false, Verbatim: false},
			Value: "b",
			URL:   ""}),
	}, optionBlockObjects)
}

type dummyClient struct {
	members          []string
	err              error
	response         url.Values
	viewResponse     *slack.ViewResponse
	Channel          *slack.Channel
	Reminder         *slack.Reminder
	AuthTestResponse *slack.AuthTestResponse
	User             *slack.User
	Channels         []slack.Channel
	NextCursor       string
}

var _ SlackClient = &dummyClient{}

func (c *dummyClient) SendMessageContext(ctx context.Context, channelID string, options ...slack.MsgOption) (_channel, _timestamp, _text string, err error) {
	_, c.response, err = slack.UnsafeApplyMsgOptions("", channelID, "", options...)
	return "", "", "", err
}

func (c *dummyClient) GetUserGroupMembersContext(ctx context.Context, userGroup string) ([]string, error) {
	return c.members, c.err
}

func (c *dummyClient) OpenViewContext(ctx context.Context, triggerID string, view slack.ModalViewRequest) (*slack.ViewResponse, error) {
	return c.viewResponse, c.err
}

func (c *dummyClient) UpdateViewContext(ctx context.Context, view slack.ModalViewRequest, externalID, hash, viewID string) (*slack.ViewResponse, error) {
	return c.viewResponse, c.err
}

func (c *dummyClient) CreateConversationContext(ctx context.Context, channelName string, isPrivate bool) (*slack.Channel, error) {
	return c.Channel, c.err
}

func (c *dummyClient) GetConversationInfoContext(ctx context.Context, channelID string, includeLocale bool) (*slack.Channel, error) {
	return c.Channel, c.err
}

func (c *dummyClient) ArchiveConversationContext(ctx context.Context, channelID string) error {
	return c.err
}

func (c *dummyClient) SetPurposeOfConversationContext(ctx context.Context, channelID, purpose string) (*slack.Channel, error) {
	return c.Channel, c.err
}

func (c *dummyClient) SetTopicOfConversationContext(ctx context.Context, channelID, topic string) (*slack.Channel, error) {
	return c.Channel, c.err
}

func (c *dummyClient) InviteUsersToConversationContext(ctx context.Context, channelID string, users ...string) (*slack.Channel, error) {
	return c.Channel, c.err
}

func (c *dummyClient) AddChannelReminder(channelID string, text string, time string) (*slack.Reminder, error) {
	return c.Reminder, c.err
}

func (c *dummyClient) GetUserInfoContext(ctx context.Context, user string) (*slack.User, error) {
	return c.User, c.err
}

func (c *dummyClient) AuthTestContext(ctx context.Context) (*slack.AuthTestResponse, error) {
	return c.AuthTestResponse, c.err
}

func (c *dummyClient) GetConversationsForUserContext(ctx context.Context, params *slack.GetConversationsForUserParameters) ([]slack.Channel, string, error) {
	return c.Channels, c.NextCursor, c.err
}
