package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// the prometheus namespace
	NS string

	SlackBotAccessToken  string
	SlackUserAccessToken string
	SlackSigningSecret   string
	SlackAdminGroupID    string
	BroadcastChannelID   string

	Addr    string
	TLSAddr string
	TLSCert string
	TLSKey  string

	IncidentEnvs           string
	IncidentRegions        string
	IncidentSeverityLevels string
	IncidentImpactLevels   string
	IncidentDocTemplateURL string
}

func FromViper(v *viper.Viper) (Config, error) {
	c := Config{}

	c.NS = v.GetString("server.prometheusNamespace")

	c.SlackBotAccessToken = v.GetString("slack.botAccessToken")
	c.SlackUserAccessToken = v.GetString("slack.userAccessToken")
	c.SlackSigningSecret = v.GetString("slack.signingSecret")
	c.SlackAdminGroupID = v.GetString("slack.adminGroupID")
	c.BroadcastChannelID = v.GetString("slack.broadcastChannelID")

	c.Addr = v.GetString("addr")
	c.TLSAddr = v.GetString("tls.addr")
	c.TLSCert = v.GetString("tls.cert")
	c.TLSKey = v.GetString("tls.key")

	c.IncidentEnvs = v.GetString("incident.environments")
	c.IncidentRegions = v.GetString("incident.regions")
	c.IncidentSeverityLevels = v.GetString("incident.severityLevels")
	c.IncidentImpactLevels = v.GetString("incident.impactLevels")
	c.IncidentDocTemplateURL = v.GetString("incidentDocTemplateURL")

	return c, nil
}
