package bot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"github.com/karl-johan-grahn/devopsbot/internal/middleware"
	"github.com/karl-johan-grahn/devopsbot/internal/wrappedcontext"
	"github.com/rs/zerolog"
	"github.com/slack-go/slack"
)

var channelNameRegex = regexp.MustCompile(`^[a-z0-9\-\_]+$`)
var incChannelNameRegex = regexp.MustCompile(`^inc_`)

const (
	alreadyInChannel   = "already_in_channel"
	valIncChName       = "%s - channel name must be non-empty, and contain only lowercase letters, numbers, hyphens, and underscores"
	valChosenIncChName = "#%s does not seem to be an incident channel"
)

// handleInteractive - a general handler for the /interactive endpoint
func (h *botHandler) handleInteractive(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	log := zerolog.Ctx(ctx)

	payload := &slack.InteractionCallback{}
	err := json.Unmarshal([]byte(r.FormValue("payload")), payload)
	if err != nil {
		err = middleware.NewHTTPError(fmt.Errorf("failed to parse interactive payload: %w", err), r)
		log.Error().Err(err).Interface("payload", payload).Send()
		w.WriteHeader(http.StatusBadRequest)
		return
	}

	switch payload.Type {
	case slack.InteractionTypeBlockActions:
		if len(payload.ActionCallback.BlockActions) == 0 {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		action := payload.ActionCallback.BlockActions[0]
		switch action.BlockID {
		case "incident_name":
			// This is triggered when pressing enter in the text input box
			if err := validateIncidentChannelName("incident_name", action.Value); err != nil {
				if uerr := h.updateView(ctx, payload, "incident_name", "declare_incident", fmt.Sprintf(valIncChName, action.Value), w); uerr != nil {
					uerr = middleware.NewHTTPError(uerr, r)
					log.Error().Err(uerr).Msg("updateView failed")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		case "incident_channel":
			channelID := action.SelectedConversation
			channel, _ := h.slackClient.GetConversationInfoContext(ctx, channelID, false)
			if err := validateChosenIncidentChannelName("incident_channel", channel.Name); err != nil {
				if uerr := h.updateView(ctx, payload, "incident_channel", "resolve_incident", fmt.Sprintf(valChosenIncChName, channel.Name), w); uerr != nil {
					uerr = middleware.NewHTTPError(uerr, r)
					log.Error().Err(uerr).Msg("updateView failed")
					w.WriteHeader(http.StatusInternalServerError)
					return
				}
			}
		default:
			log.Error().Str("BlockID", action.BlockID).Msg("unknown BlockID")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	case slack.InteractionTypeViewSubmission:
		callbackID := payload.View.CallbackID
		if callbackID == "" {
			log.Error().Msg("callbackID empty")
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		switch callbackID {
		case "declare_incident":
			if err := h.declareIncident(ctx, payload, w); err != nil {
				err = middleware.NewHTTPError(err, r)
				log.Error().Err(err).Msg("declareIncident failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		case "resolve_incident":
			if err := h.resolveIncident(ctx, payload, w); err != nil {
				err = middleware.NewHTTPError(err, r)
				log.Error().Err(err).Msg("resolveIncident failed")
				w.WriteHeader(http.StatusInternalServerError)
				return
			}
		default:
			log.Error().Str("callbackID", callbackID).Msg("unknown callbackID")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	default:
		log.Warn().Str("payload_type", string(payload.Type)).Msg("unknown interactive payload type")
		w.WriteHeader(http.StatusNotFound)
	}
}

func (h *botHandler) updateView(ctx context.Context, payload *slack.InteractionCallback, blockID string, callbackID string, updatedText string, w http.ResponseWriter) error {
	log := zerolog.Ctx(ctx)
	for _, b := range payload.View.Blocks.BlockSet {
		if input, ok := b.(*slack.InputBlock); ok && input.BlockID == blockID {
			input.Hint = slack.NewTextBlockObject(slack.PlainTextType, updatedText, false, false)
			input.DispatchAction = true
		}
	}
	mvr := slack.ModalViewRequest{
		Type:       payload.View.Type,
		Title:      payload.View.Title,
		Close:      payload.View.Close,
		Submit:     payload.View.Submit,
		Blocks:     payload.View.Blocks,
		CallbackID: callbackID,
	}
	_, err := h.slackClient.UpdateViewContext(ctx, mvr, payload.View.ExternalID, payload.View.Hash, payload.View.ID)
	if err != nil {
		log.Error().Err(err).Msg("Error updating view")
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	return nil
}

type inputParams struct {
	incidentChannelName          string
	incidentSecurityRelated      bool
	incidentResponder            string
	incidentCommander            string
	incidentInvitees             []string
	incidentEnvironmentsAffected string
	incidentRegionsAffected      string
	incidentSummary              string
	incidentDeclarer             string
	broadcastChannel             string
}

type validationError struct {
	errors map[string]string
}

// Error - conform to the error interface
func (e *validationError) Error() string {
	badFields := make([]string, len(e.errors))
	i := 0
	for _, k := range e.errors {
		badFields[i] = k
		i++
	}
	return fmt.Sprintf("validation errors with field(s): %s", strings.Join(badFields, ", "))
}

// validatePayload - validate incident payload
func validatePayload(ctx context.Context, payload *slack.InteractionCallback) error {
	incidentChannelName := createChannelName(payload.View.State.Values["incident_name"]["incident_name"].Value)
	return validateIncidentChannelName("incident_name", incidentChannelName)
}

// declareIncident - general handler for incident commands
func (h *botHandler) declareIncident(ctx context.Context, payload *slack.InteractionCallback, w http.ResponseWriter) error {
	if err := validatePayload(ctx, payload); err != nil {
		var verr *validationError
		if errors.As(err, &verr) {
			return postErrorResponse(ctx, verr.errors, w)
		}
		return err
	}
	incidentChannelName := createChannelName(payload.View.State.Values["incident_name"]["incident_name"].Value)
	incidentEnvironmentsAffected := make([]string, len(payload.View.State.Values["incident_environment_affected"]["incident_environment_affected"].SelectedOptions))
	for i, e := range payload.View.State.Values["incident_environment_affected"]["incident_environment_affected"].SelectedOptions {
		incidentEnvironmentsAffected[i] = e.Value
	}
	incidentRegionsAffected := make([]string, len(payload.View.State.Values["incident_region_affected"]["incident_region_affected"].SelectedOptions))
	for i, r := range payload.View.State.Values["incident_region_affected"]["incident_region_affected"].SelectedOptions {
		incidentRegionsAffected[i] = r.Value
	}
	inputParams := &inputParams{
		incidentChannelName:          incidentChannelName,
		incidentSecurityRelated:      payload.View.State.Values["security_incident"]["security_incident"].SelectedOption.Value == "yes",
		incidentResponder:            payload.View.State.Values["incident_responder"]["incident_responder"].SelectedUser,
		incidentCommander:            payload.View.State.Values["incident_commander"]["incident_commander"].SelectedUser,
		incidentInvitees:             payload.View.State.Values["incident_invitees"]["incident_invitees"].SelectedUsers,
		incidentEnvironmentsAffected: strings.Join(incidentEnvironmentsAffected, ", "),
		incidentRegionsAffected:      strings.Join(incidentRegionsAffected, ", "),
		incidentSummary:              payload.View.State.Values["incident_summary"]["incident_summary"].Value,
		incidentDeclarer:             payload.User.ID,
		broadcastChannel:             payload.View.State.Values["broadcast_channel"]["broadcast_channel"].SelectedConversation,
	}

	// Create channel - should be done here because it will update the modal if there are errors
	incidentChannel, err := h.slackClient.CreateConversationContext(ctx, incidentChannelName, inputParams.incidentSecurityRelated)
	if err != nil {
		errorMessage := createUserFriendlyConversationError(err)
		return postErrorResponse(ctx, map[string]string{
			"incident_name": fmt.Sprintf("%s - %s", incidentChannelName, errorMessage),
		}, w)
	}

	// Invite devopsbot to broad cast channel so it is able to send messages there.
	// If it is already in the channel, don't return the error, just continue.
	authTestResp, _ := h.slackClient.AuthTestContext(ctx)
	if _, err := h.slackClient.InviteUsersToConversationContext(ctx, inputParams.broadcastChannel, authTestResp.UserID); err != nil {
		if err.Error() != alreadyInChannel {
			return postErrorResponse(ctx, map[string]string{
				"broadcast_channel": fmt.Sprintf("%s - %s", inputParams.broadcastChannel, err),
			}, w)
		}
	}
	// Invite devopsbot to incident channel so it is able to perform actions there
	if _, err := h.slackClient.InviteUsersToConversationContext(ctx, incidentChannel.ID, authTestResp.UserID); err != nil {
		if err.Error() != alreadyInChannel {
			return postErrorResponse(ctx, map[string]string{
				"incident_name": fmt.Sprintf("%s - %s", incidentChannel.ID, err),
			}, w)
		}
	}

	if err := h.sendMessage(ctx, inputParams.broadcastChannel, slack.MsgOptionPostEphemeral(inputParams.incidentDeclarer),
		slack.MsgOptionText(fmt.Sprintf("Started work on declaring the incident <#%s>", incidentChannel.ID), false)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.WriteHeader(http.StatusAccepted)

	// Do the rest via goroutine
	ctx = wrappedcontext.WrapContextValues(context.Background(), ctx)
	go h.doIncidentTasks(ctx, inputParams, incidentChannel)

	return nil
}

// doIncidentTasks - do incident creation tasks asynchronously
func (h *botHandler) doIncidentTasks(ctx context.Context, params *inputParams, incidentChannel *slack.Channel) {
	log := zerolog.Ctx(ctx)
	const sendError = "Could not send failure message"
	var securityMessage string
	if params.incidentSecurityRelated {
		securityMessage = "This is a security related incident - available by invitation only"
	} else {
		securityMessage = ""
	}
	// Set channel purpose and topic
	overview := fmt.Sprintf("*Summary:* %s\n"+
		"*Environment affected:* %s\n"+
		"*Region affected:* %s\n"+
		"*Responder:* <@%s>\n"+
		"*Commander:* <@%s>\n\n"+
		"Declared by: <@%s>\n"+
		securityMessage,
		params.incidentSummary, params.incidentEnvironmentsAffected, params.incidentRegionsAffected,
		params.incidentResponder, params.incidentCommander, params.incidentDeclarer)
	if _, err := h.slackClient.SetPurposeOfConversationContext(ctx, incidentChannel.ID, overview); err != nil {
		if sendErr := h.sendMessage(ctx, params.broadcastChannel, slack.MsgOptionPostEphemeral(params.incidentDeclarer),
			slack.MsgOptionText(fmt.Sprintf("Failed to set purpose for incident channel: %s", err.Error()), false)); sendErr != nil {
			log.Error().Err(sendErr).Msg(sendError)
			return
		}
	}
	if _, err := h.slackClient.SetTopicOfConversationContext(ctx, incidentChannel.ID, overview); err != nil {
		if sendErr := h.sendMessage(ctx, params.broadcastChannel, slack.MsgOptionPostEphemeral(params.incidentDeclarer),
			slack.MsgOptionText(fmt.Sprintf("Failed to set topic for incident channel: %s", err.Error()), false)); sendErr != nil {
			log.Error().Err(sendErr).Msg(sendError)
			return
		}
	}
	// Add invitees to channel - the InviteUsersToConversationContext method
	// does not accept group as user so have to specify users individually
	if _, err := h.slackClient.InviteUsersToConversationContext(ctx, incidentChannel.ID, params.incidentInvitees...); err != nil {
		if err.Error() != alreadyInChannel {
			if sendErr := h.sendMessage(ctx, params.broadcastChannel, slack.MsgOptionPostEphemeral(params.incidentDeclarer),
				slack.MsgOptionText(fmt.Sprintf("Failed to add invitees to incident channel: %s", err.Error()), false)); sendErr != nil {
				log.Error().Err(sendErr).Msg(sendError)
				return
			}
		}
	}
	// Inform about incident
	if err := h.sendMessage(ctx, params.broadcastChannel,
		slack.MsgOptionText(fmt.Sprintf(":rotating_siren: An incident has been declared by <@%s>\n"+
			"*Summary:* %s\n"+
			"*Environment affected:* %s\n"+
			"*Region affected:* %s\n"+
			"*Responder:* <@%s>\n"+
			"*Commander:* <@%s>\n"+
			"*Incident channel:* <#%s>\n"+
			securityMessage,
			params.incidentDeclarer, params.incidentSummary, params.incidentEnvironmentsAffected,
			params.incidentRegionsAffected, params.incidentResponder, params.incidentCommander,
			incidentChannel.ID), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
	// Send message about starting slack call
	if err := h.sendMessage(ctx, incidentChannel.ID,
		slack.MsgOptionText(fmt.Sprintf("IC <@%s>: Start a Teams call with the command `/teams-calls meeting %s`", params.incidentCommander, params.incidentChannelName), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
	// Send message about starting incident doc
	if err := h.sendMessage(ctx, incidentChannel.ID,
		slack.MsgOptionText(fmt.Sprintf("IC <@%s>: Start the incident doc by using [this template](%s)", params.incidentCommander, h.opts.IncidentDocTemplateURL), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
	// Add channel reminder about updating about progress
	if _, err := h.slackClient.AddChannelReminder(
		incidentChannel.ID,
		fmt.Sprintf("Reminder for IC <@%s>: Update progress about the incident in <#%s>", params.incidentCommander, params.broadcastChannel),
		"Every 30 min"); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
	// Inform about being done with declaring incident
	if err := h.sendMessage(ctx, params.broadcastChannel, slack.MsgOptionPostEphemeral(params.incidentDeclarer),
		slack.MsgOptionText(fmt.Sprintf("Finished declaring the incident %s", params.incidentChannelName), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
}

type resolveParams struct {
	// Incident channel ID
	incidentChannel string
	// Incident resolution
	incidentResolution string
	// Should the incident channel be archived
	incidentArchive bool
	// Whow is resolving the incident
	incidentResolver string
	// Broadcast channel to announce resolvement
	broadcastChannel string
}

// resolveIncident - handler for resolving incidents
func (h *botHandler) resolveIncident(ctx context.Context, payload *slack.InteractionCallback, w http.ResponseWriter) error {
	incidentChannelID := payload.View.State.Values["incident_channel"]["incident_channel"].SelectedConversation
	incChannel, _ := h.slackClient.GetConversationInfoContext(ctx, incidentChannelID, false)
	if err := validateChosenIncidentChannelName("incident_channel", incChannel.Name); err != nil {
		var verr *validationError
		if errors.As(err, &verr) {
			return postErrorResponse(ctx, verr.errors, w)
		}
		return err
	}
	resolveParams := &resolveParams{
		incidentChannel:    incidentChannelID,
		incidentResolution: payload.View.State.Values["resolution"]["resolution"].Value,
		incidentArchive:    payload.View.State.Values["archive_choice"]["archive_choice"].SelectedOption.Value == "Yes",
		incidentResolver:   payload.User.ID,
		broadcastChannel:   payload.View.State.Values["broadcast_channel"]["broadcast_channel"].SelectedConversation,
	}

	// Invite devopsbot to broad cast channel so it is able to send messages there.
	// If it is already in the channel, don't return the error, just continue.
	authTestResp, _ := h.slackClient.AuthTestContext(ctx)
	if _, err := h.slackClient.InviteUsersToConversationContext(ctx, resolveParams.broadcastChannel, authTestResp.UserID); err != nil {
		if err.Error() != alreadyInChannel {
			return postErrorResponse(ctx, map[string]string{
				"broadcast_channel": fmt.Sprintf("%s - %s", resolveParams.broadcastChannel, err),
			}, w)
		}
	}

	if err := h.sendMessage(ctx, resolveParams.broadcastChannel, slack.MsgOptionPostEphemeral(resolveParams.incidentResolver),
		slack.MsgOptionText(fmt.Sprintf("Started work on resolving the incident <#%s>", incChannel.Name), false)); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return err
	}
	w.WriteHeader(http.StatusAccepted)

	// Do the rest via goroutine
	ctx = wrappedcontext.WrapContextValues(context.Background(), ctx)
	go h.doResolveTasks(ctx, resolveParams)

	return nil
}

func (h *botHandler) doResolveTasks(ctx context.Context, params *resolveParams) {
	log := zerolog.Ctx(ctx)
	const sendError = "Could not send failure message"
	// Inform about resolution
	if err := h.sendMessage(ctx, params.broadcastChannel,
		slack.MsgOptionText(fmt.Sprintf(":white_check_mark: The incident <#%s> has been resolved!\n"+
			"*Resolution:* %s",
			params.incidentChannel, params.incidentResolution), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
	if params.incidentArchive {
		if err := h.slackClient.ArchiveConversationContext(ctx, params.incidentChannel); err != nil {
			log.Error().Err(err).Msg(sendError)
			return
		}
	}
	// Inform about being done with declaring incident
	if err := h.sendMessage(ctx, params.broadcastChannel, slack.MsgOptionPostEphemeral(params.incidentResolver),
		slack.MsgOptionText(fmt.Sprintf("Finished resolving the incident <#%s>", params.incidentChannel), false)); err != nil {
		log.Error().Err(err).Msg(sendError)
		return
	}
}

// sendMessage - a simplified way to send a message
func (h *botHandler) sendMessage(ctx context.Context, channelID string, options ...slack.MsgOption) error {
	log := zerolog.Ctx(ctx)
	ch, ts, txt, err := h.slackClient.SendMessageContext(ctx, channelID, options...)
	if err != nil {
		err = fmt.Errorf("failed to send message: %w", err)
		log.Error().Err(err).Str("channel", ch).Str("message_timestamp", ts).Str("message_text", txt).Send()
		return err
	}
	return nil
}

func postErrorResponse(ctx context.Context, verr map[string]string, w http.ResponseWriter) error {
	log := zerolog.Ctx(ctx)
	errorResponse := slack.NewErrorsViewSubmissionResponse(verr)
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	enc := json.NewEncoder(w)
	err := enc.Encode(errorResponse)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	errorResponseData, err := json.Marshal(errorResponse)
	if err != nil {
		log.Err(err).Send()
		return err
	}
	log.Warn().Str("errorResponseData", string(errorResponseData)).Msg("Modal validation error")
	return nil
}

// createChannelName - create incident channel name based on input field
func createChannelName(s string) string {
	return fmt.Sprintf("inc_%s_%s", s, strings.ToLower(time.Now().Format("Jan_2")))
}

// createUserFriendlyConversationError - Map https://api.slack.com/methods/conversations.create error codes to user friendly messages
func createUserFriendlyConversationError(err error) error {
	if err.Error() == "name_taken" {
		return fmt.Errorf("channel with given name already exists: %w", err)
	}
	return err
}

// validateIncidentChannelName - validate channel name according to slack rules in https://api.slack.com/methods/conversations.create
func validateIncidentChannelName(field string, n string) error {
	errorMessage := make(map[string]string)
	if !channelNameRegex.MatchString(n) {
		errorMessage[field] = fmt.Sprintf(valIncChName, n)
		return &validationError{
			errors: errorMessage,
		}
	}
	return nil
}

func validateChosenIncidentChannelName(field string, n string) error {
	errorMessage := make(map[string]string)
	if !incChannelNameRegex.MatchString(n) {
		errorMessage[field] = fmt.Sprintf(valChosenIncChName, n)
		return &validationError{
			errors: errorMessage,
		}
	}
	return nil
}
