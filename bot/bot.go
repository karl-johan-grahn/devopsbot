package bot

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"sync"

	"github.com/nicksnyder/go-i18n/v2/i18n"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
	"golang.org/x/text/language"
)

type botHandler struct {
	slackClient SlackClient
	opts        Opts

	admins *ugMembers
}

type ugMembers struct {
	sync.RWMutex

	members []string
}

type Opts struct {
	// UserAccessToken - the Slack user access token
	UserAccessToken string
	// SigningSecret - the signing secret from the Slack app config
	SigningSecret string
	// BroadcastChannelID - the ID of the Slack channel the bot will broadcast in
	BroadcastChannelID string
	// AdminGroupID - the ID of the user group that will have admin rights to interact with the bot
	AdminGroupID string
	// IncidentDocTemplateURL - the URL of the incident document template
	IncidentDocTemplateURL string
	// IncidentEnvs - the environments that could possibly be affected
	IncidentEnvs string
	// IncidentRegions - the regions that could possibly be affected
	IncidentRegions string
	// IncidentSeverityLevels - the possible severity levels of an incident
	IncidentSeverityLevels string
	// IncidentImpactLevels - the possible impact levels of an incident
	IncidentImpactLevels string
	// Localizer - the localizer to use for the set of language preferences
	Localizer *i18n.Localizer
}

// NewBot - create a new bot handler
func NewBot(slackClient SlackClient, opts Opts) http.Handler {
	h := &botHandler{
		slackClient: slackClient,
		opts:        opts,
		admins:      &ugMembers{},
	}

	m := http.NewServeMux()
	m.HandleFunc("/command", h.handleCommand)
	m.HandleFunc("/interactive", h.handleInteractive)

	return mwVerify(h.opts.SigningSecret, m)
}

func (h *botHandler) handleCommand(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := zerolog.Ctx(ctx)

	cmd, err := slack.SlashCommandParse(r)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = h.errorResponse(ctx, w, cmd, fmt.Sprintf("failed to parse command: %s", err), err)
		return
	}

	*log = log.With().
		Str("user_id", cmd.UserID).
		Str("user_name", cmd.UserName).
		Str("channel_id", cmd.ChannelID).
		Str("channel_name", cmd.ChannelName).
		Str("command", cmd.Command).
		Logger()
	ctx = log.WithContext(ctx)
	bundle := i18n.NewBundle(language.English)
	bundle.RegisterUnmarshalFunc("json", json.Unmarshal)
	bundle.MustLoadMessageFile("active.en.json")
	bundle.MustLoadMessageFile("active.fr.json")
	user, err := h.slackClient.GetUserInfoContext(ctx, cmd.UserID)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		_ = h.errorResponse(ctx, w, cmd, fmt.Sprintf("failed to get user info: %s", err), err)
		return
	}
	// Look up strings in English first as a fallback mechanism
	h.opts.Localizer = i18n.NewLocalizer(bundle, language.English.String(), user.Locale)

	switch {
	case strings.HasSuffix(cmd.Command, "devopsbot"):
		log.Info().Str("text", cmd.Text).Send()

		// split the command text into the action and the rest
		parts := strings.SplitN(cmd.Text, " ", 2)

		switch parts[0] {
		case "incident":
			err = h.cmdIncident(ctx, w, cmd)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = h.errorResponse(ctx, w, cmd, fmt.Sprintf("cmdIncident failed: %s", err), err)
			}
			return
		case "resolve":
			err = h.cmdResolveIncident(ctx, w, cmd)
			if err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				_ = h.errorResponse(ctx, w, cmd, fmt.Sprintf("cmdResolveIncident failed: %s", err), err)
			}
			return
		default:
			if err := h.respond(ctx, cmd.ResponseURL, cmd.UserID, slack.ResponseTypeEphemeral,
				slack.MsgOptionText(h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
					DefaultMessage: &i18n.Message{
						ID: "HelpMessage",
						Other: "These are the available commands:\n" +
							"> `/devopsbot help` - Get this help\n" +
							"> `/devopsbot incident` - Declare an incident\n" +
							"> `/devopsbot resolve` - Resolve an incident"},
				}), false),
				slack.MsgOptionAttachments(),
			); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
			w.WriteHeader(http.StatusOK)
			return
		}
	default:
		log.Warn().Msg("unknown command")
		w.WriteHeader(http.StatusNotFound)
	}
}

// cmdIncident - general handler for /devops incident commands
func (h *botHandler) cmdIncident(ctx context.Context, w http.ResponseWriter, cmd slack.SlashCommand) error {
	titleText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "DeclareNewIncident",
				Other: "Declare a new incident"},
		}), false, false)
	closeText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Cancel",
				Other: "Cancel"},
		}), false, false)
	submitText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "DeclareIncident",
				Other: "Declare incident"},
		}), false, false)

	contextText := slack.NewTextBlockObject(slack.MarkdownType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "IncidentCreationDescription",
				Other: "This will create a new incident Slack channel, and notify about the incident in a broadcast channel. This incident response system is based on the Incident Command System."},
		}), false, false)
	contextBlock := slack.NewContextBlock("context", contextText)

	broadcastChLabel := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "BroadcastChannel",
				Other: "Broadcast channel"},
		}), false, false)
	// TODO: Only get one page of results for now, implement pagination later if needed
	authTestResp, _ := h.slackClient.AuthTestContext(ctx)
	channels, _, err := h.slackClient.GetConversationsForUserContext(ctx, &slack.GetConversationsForUserParameters{
		UserID:          authTestResp.UserID,
		Limit:           100,
		ExcludeArchived: true,
	})
	if err != nil {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("Failed to get conversations for bot: %s", err), err)
	}
	if channels == nil {
		return h.errorResponse(ctx, w, cmd, "Bot must be added to a channel for broadcasting messages", nil)
	}
	channelIDs := []string{}
	var botInBroadcastChannel = false
	for i := range channels {
		channelIDs = append(channelIDs, channels[i].ID)
		if channels[i].ID == h.opts.BroadcastChannelID {
			botInBroadcastChannel = true
		}
	}
	if !botInBroadcastChannel {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("The bot is not part of the configured broadcast channel <#%s>, invite it there first", h.opts.BroadcastChannelID), nil)
	}
	broadcastChannel, err := h.slackClient.GetConversationInfoContext(ctx, h.opts.BroadcastChannelID, false)
	if broadcastChannel.IsArchived {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("The configured broadcast channel <#%s> is archived, update the configuration to use an open broadcast channel", h.opts.BroadcastChannelID), err)
	}
	botChannels := createOptionBlockObjects(channelIDs, "channel")
	broadcastChOption := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, nil, "broadcast_channel", botChannels...)
	initialChannelLabel := slack.NewTextBlockObject(slack.PlainTextType,
		fmt.Sprintf("<#%s>", h.opts.BroadcastChannelID), false, false)
	broadcastChOption.InitialOption = slack.NewOptionBlockObject(h.opts.BroadcastChannelID, initialChannelLabel, nil)
	broadcastChHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "BroadcastChannelHint",
				Other: "The channels listed are the ones that the bot has been added to as a user"},
		}), false, false)
	broadcastChBlock := slack.NewInputBlock("broadcast_channel", broadcastChLabel, broadcastChHint, broadcastChOption)

	// Only the inputs in input blocks will be included in view_submission’s view.state.values: https://slack.dev/java-slack-sdk/guides/modals
	incidentNameText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "IncidentName",
				Other: "Incident name"},
		}), false, false)
	incidentNameElement := slack.NewPlainTextInputBlockElement(incidentNameText, "incident_name")
	// Name will be prefixed with "inc_" and postfixed with "_<date>" so keep it shorter than the maximum 80 characters
	incidentNameElement.MaxLength = 60
	incidentNameElement.DispatchActionConfig = &slack.DispatchActionConfig{
		TriggerActionsOn: []string{"on_character_entered"},
	}
	incidentNameHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "IncidentNameHint",
				Other: "Incident names may only contain lowercase letters, numbers, hyphens, and underscores, and must be 60 characters or less"},
		}), false, false)
	incidentNameBlock := slack.NewInputBlock("incident_name", incidentNameText, incidentNameHint, incidentNameElement)
	incidentNameBlock.DispatchAction = true

	securityIncHeading := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "SecurityIncident",
				Other: "Security Incident"},
		}), false, false)
	securityIncLabel := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "SecurityIncidentLabel",
				Other: "Mark to make incident channel private"},
		}), false, false)
	securityOptionBlockObject := slack.NewOptionBlockObject("yes", securityIncLabel, nil)
	securityOptionsBlock := slack.NewCheckboxGroupsBlockElement("security_incident", securityOptionBlockObject)
	securityBlock := slack.NewInputBlock("security_incident", securityIncHeading, nil, securityOptionsBlock)
	securityBlock.Optional = true

	responderText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Responder",
				Other: "Responder"},
		}), false, false)
	responderOption := slack.NewOptionsSelectBlockElement(slack.OptTypeUser, responderText, "incident_responder")
	responderHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "ResponderHint",
				Other: "The responder leads the work of resolving the incident"},
		}), false, false)
	responderBlock := slack.NewInputBlock("incident_responder", responderText, responderHint, responderOption)

	commanderText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Commander",
				Other: "Commander"},
		}), false, false)
	commanderOption := slack.NewOptionsSelectBlockElement(slack.OptTypeUser, commanderText, "incident_commander")
	commanderHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "CommanderHint",
				Other: "The incident commander coordinates, communicates, and controls the response"},
		}), false, false)
	commanderBlock := slack.NewInputBlock("incident_commander", commanderText, commanderHint, commanderOption)

	envTxt := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Environment",
				Other: "Environment"},
		}), false, false)
	var envs []string
	if err := json.Unmarshal([]byte(h.opts.IncidentEnvs), &envs); err != nil {
		return h.errorResponse(ctx, w, cmd, "Failed to unmarshal incident environments", err)
	}
	envOptions := createOptionBlockObjects(envs, "")
	envOptionsBlock := slack.NewCheckboxGroupsBlockElement("incident_environment_affected", envOptions...)
	environmentBlock := slack.NewInputBlock("incident_environment_affected", envTxt, nil, envOptionsBlock)

	regionTxt := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Region",
				Other: "Region"},
		}), false, false)
	var regions []string
	if err := json.Unmarshal([]byte(h.opts.IncidentRegions), &regions); err != nil {
		return h.errorResponse(ctx, w, cmd, "Failed to unmarshal incident regions", err)
	}
	regionOptions := createOptionBlockObjects(regions, "")
	regionOptionsBlock := slack.NewCheckboxGroupsBlockElement("incident_region_affected", regionOptions...)
	regionBlock := slack.NewInputBlock("incident_region_affected", regionTxt, nil, regionOptionsBlock)

	severityTxt := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Severity",
				Other: "Severity"},
		}), false, false)
	var severityLevels []string
	if err := json.Unmarshal([]byte(h.opts.IncidentSeverityLevels), &severityLevels); err != nil {
		return h.errorResponse(ctx, w, cmd, "Failed to unmarshal incident severity levels", err)
	}
	severityOptions := createOptionBlockObjects(severityLevels, "")
	severityOptionsBlock := slack.NewRadioButtonsBlockElement("incident_severity_level", severityOptions...)
	severityBlock := slack.NewInputBlock("incident_severity_level", severityTxt, nil, severityOptionsBlock)

	impactTxt := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Impact",
				Other: "Impact"},
		}), false, false)
	var impactLevels []string
	if err := json.Unmarshal([]byte(h.opts.IncidentImpactLevels), &impactLevels); err != nil {
		return h.errorResponse(ctx, w, cmd, "Failed to unmarshal incident impact levels", err)
	}
	impactOptions := createOptionBlockObjects(impactLevels, "")
	impactOptionsBlock := slack.NewRadioButtonsBlockElement("incident_impact_level", impactOptions...)
	impactBlock := slack.NewInputBlock("incident_impact_level", impactTxt, nil, impactOptionsBlock)

	summaryText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "IncidentSummary",
				Other: "Incident summary"},
		}), false, false)
	summaryElement := slack.NewPlainTextInputBlockElement(summaryText, "incident_summary")
	// Set an arbitrary max length to avoid prose summary
	summaryElement.MaxLength = 200
	summaryElement.Multiline = true
	summaryBlock := slack.NewInputBlock("incident_summary", summaryText, nil, summaryElement)

	inviteeLabel := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Invitees",
				Other: "Invitees"},
		}), false, false)
	inviteeOption := slack.NewOptionsSelectBlockElement(slack.MultiOptTypeUser, inviteeLabel, "incident_invitees")
	inviteeBlock := slack.NewInputBlock("incident_invitees", inviteeLabel, nil, inviteeOption)
	inviteeBlock.Optional = true

	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			contextBlock,
			broadcastChBlock,
			incidentNameBlock,
			securityBlock,
			responderBlock,
			commanderBlock,
			environmentBlock,
			regionBlock,
			severityBlock,
			impactBlock,
			summaryBlock,
			inviteeBlock,
		},
	}

	var modalVReq slack.ModalViewRequest
	modalVReq.Type = slack.ViewType("modal")
	modalVReq.Title = titleText
	modalVReq.Close = closeText
	modalVReq.Submit = submitText
	modalVReq.Blocks = blocks
	modalVReq.ClearOnClose = true
	modalVReq.CallbackID = "declare_incident"

	_, err = h.slackClient.OpenViewContext(ctx, cmd.TriggerID, modalVReq)
	if err != nil {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("Error opening view: %s", err), err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

// cmdResolveIncident - handler for resolving incident
func (h *botHandler) cmdResolveIncident(ctx context.Context, w http.ResponseWriter, cmd slack.SlashCommand) error {
	titleText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "ResolveAnIncident",
				Other: "Resolve an incident"},
		}), false, false)
	closeText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Cancel",
				Other: "Cancel"},
		}), false, false)
	submitText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "ResolveIncident",
				Other: "Resolve incident"},
		}), false, false)

	contextText := slack.NewTextBlockObject(slack.MarkdownType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "ResolveIncidentDescription",
				Other: "This will resolve an incident and notify about the resolution in a broadcast channel"},
		}), false, false)
	contextBlock := slack.NewContextBlock("context", contextText)

	broadcastChLabel := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "BroadcastChannel",
				Other: "Broadcast channel"},
		}), false, false)
	// TODO: Only get one page of results for now, implement pagination later if needed
	authTestResp, _ := h.slackClient.AuthTestContext(ctx)
	channels, _, err := h.slackClient.GetConversationsForUserContext(ctx, &slack.GetConversationsForUserParameters{
		UserID:          authTestResp.UserID,
		Limit:           100,
		ExcludeArchived: true,
	})
	if err != nil {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("Failed to get conversations for bot: %s", err), err)
	}
	if channels == nil {
		return h.errorResponse(ctx, w, cmd, "Bot must be added to a channel for broadcasting messages", nil)
	}
	channelIDs := []string{}
	var botInBroadcastChannel = false
	for i := range channels {
		channelIDs = append(channelIDs, channels[i].ID)
		if channels[i].ID == h.opts.BroadcastChannelID {
			botInBroadcastChannel = true
		}
	}
	if !botInBroadcastChannel {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("The bot is not part of the configured broadcast channel <#%s>, invite it there first", h.opts.BroadcastChannelID), nil)
	}
	broadcastChannel, err := h.slackClient.GetConversationInfoContext(ctx, h.opts.BroadcastChannelID, false)
	if broadcastChannel.IsArchived {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("The configured broadcast channel <#%s> is archived, update the configuration to use an open broadcast channel", h.opts.BroadcastChannelID), err)
	}
	botChannels := createOptionBlockObjects(channelIDs, "channel")
	broadcastChOption := slack.NewOptionsSelectBlockElement(slack.OptTypeStatic, nil, "broadcast_channel", botChannels...)
	initialChannelLabel := slack.NewTextBlockObject(slack.PlainTextType,
		fmt.Sprintf("<#%s>", h.opts.BroadcastChannelID), false, false)
	broadcastChOption.InitialOption = slack.NewOptionBlockObject(h.opts.BroadcastChannelID, initialChannelLabel, nil)
	broadcastChHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "BroadcastChannelHint",
				Other: "The channels listed are the ones that the bot has been added to as a user"},
		}), false, false)
	broadcastChBlock := slack.NewInputBlock("broadcast_channel", broadcastChLabel, broadcastChHint, broadcastChOption)

	incChanText := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Incident",
				Other: "Incident"},
		}), false, false)
	incChanOption := slack.NewOptionsSelectBlockElement(slack.OptTypeConversations, incChanText, "incident_channel")
	incChanOption.DefaultToCurrentConversation = true
	incChanOption.Filter = &slack.SelectBlockElementFilter{
		Include:                       []string{"public"},
		ExcludeExternalSharedChannels: false,
		ExcludeBotUsers:               false,
	}
	incChanHint := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "IncidentChannelNamePattern",
				Other: "Choose a channel that starts with 'inc_'"},
		}), false, false)
	incChanBlock := slack.NewInputBlock("incident_channel", incChanText, incChanHint, incChanOption)
	incChanBlock.DispatchAction = true

	archiveTxt := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "ArchiveIncidentChannel",
				Other: "Archive incident channel"},
		}), false, false)
	archiveOptions := createOptionBlockObjects([]string{
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Yes",
				Other: "Yes"},
		}),
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "No",
				Other: "No"},
		})}, "")
	archiveOptionsBlock := slack.NewRadioButtonsBlockElement("archive_choice", archiveOptions...)
	archiveBlock := slack.NewInputBlock("archive_choice", archiveTxt, nil, archiveOptionsBlock)

	resolutionLabel := slack.NewTextBlockObject(slack.PlainTextType,
		h.opts.Localizer.MustLocalize(&i18n.LocalizeConfig{
			DefaultMessage: &i18n.Message{
				ID:    "Resolution",
				Other: "Resolution"},
		}), false, false)
	resolutionElement := slack.NewPlainTextInputBlockElement(resolutionLabel, "resolution")
	resolutionElement.MaxLength = 200
	resolutionElement.Multiline = true
	resolutionBlock := slack.NewInputBlock("resolution", resolutionLabel, nil, resolutionElement)

	blocks := slack.Blocks{
		BlockSet: []slack.Block{
			contextBlock,
			broadcastChBlock,
			incChanBlock,
			archiveBlock,
			resolutionBlock,
		},
	}

	var modalVReq slack.ModalViewRequest
	modalVReq.Type = slack.ViewType("modal")
	modalVReq.Title = titleText
	modalVReq.Close = closeText
	modalVReq.Submit = submitText
	modalVReq.Blocks = blocks
	modalVReq.ClearOnClose = true
	modalVReq.CallbackID = "resolve_incident"

	_, err = h.slackClient.OpenViewContext(ctx, cmd.TriggerID, modalVReq)
	if err != nil {
		return h.errorResponse(ctx, w, cmd, fmt.Sprintf("Error opening view: %s", err), err)
	}

	w.WriteHeader(http.StatusOK)
	return nil
}

// createOptionBlockObjects - utility function for generating option block objects
func createOptionBlockObjects(options []string, optionType string) []*slack.OptionBlockObject {
	optionBlockObjects := make([]*slack.OptionBlockObject, 0, len(options))
	var text string
	for _, o := range options {
		if optionType == "channel" {
			text = fmt.Sprintf("<#%s>", o)
		} else {
			text = o
		}
		optionText := slack.NewTextBlockObject(slack.PlainTextType, text, false, false)
		optionBlockObjects = append(optionBlockObjects, slack.NewOptionBlockObject(o, optionText, nil))
	}
	return optionBlockObjects
}

// errorResponse - send ephemeral error response via Slack UI
func (h *botHandler) errorResponse(ctx context.Context, w http.ResponseWriter, cmd slack.SlashCommand, errorText string, err error) error {
	if sendErr := h.respond(ctx, cmd.ResponseURL, cmd.UserID, slack.ResponseTypeEphemeral,
		slack.MsgOptionText(errorText, false),
		slack.MsgOptionAttachments(),
	); sendErr != nil {
		return sendErr
	}
	return err
}

// respond - a simplified way to respond
func (h *botHandler) respond(ctx context.Context, responseURL, channelID, responseType string, options ...slack.MsgOption) error {
	log := zerolog.Ctx(ctx)
	options = append(options, slack.MsgOptionResponseURL(responseURL, responseType))
	ch, ts, txt, err := h.slackClient.SendMessageContext(ctx, channelID, options...)
	if err != nil {
		err = fmt.Errorf("failed to respond with message: %w", err)
		log.Error().Err(err).Str("channel", ch).Str("message_timestamp", ts).Str("message_text", txt).Send()
		return err
	}
	return nil
}

func (h *botHandler) isAdmin(ctx context.Context, userID string) bool {
	log := zerolog.Ctx(ctx)

	if len(h.admins.members) == 0 {
		log.Info().Msg("User group empty, updating internal list")
		err := h.updateAdmins(ctx, h.opts.AdminGroupID)
		if err != nil {
			log.Error().Err(err).Msg("Failed to get user group members")
			return false
		}
	}

	h.admins.RLock()
	defer h.admins.RUnlock()
	for _, m := range h.admins.members {
		if userID == m {
			return true
		}
	}
	return false
}

// updateAdmins - update the bot's internal admin list
func (h *botHandler) updateAdmins(ctx context.Context, userGroup string) error {
	log := zerolog.Ctx(ctx)

	members, err := h.slackClient.GetUserGroupMembersContext(ctx, userGroup)
	if err != nil {
		return fmt.Errorf("failed to get members of group %q: %w", userGroup, err)
	}
	log.Info().Strs("members", members).Msg("Updating internal user group list")

	h.admins.Lock()
	defer h.admins.Unlock()
	h.admins.members = members
	return nil
}
