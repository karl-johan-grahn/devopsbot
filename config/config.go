package config

import (
	"github.com/spf13/viper"
)

type Config struct {
	// the prometheus namespace
	NS string

	SlackAccessToken   string
	SlackSigningSecret string
	SlackAdminGroupID  string
	SlackChannelID     string

	Addr    string
	TLSAddr string
	TLSCert string
	TLSKey  string

	IncidentEnvs           string
	IncidentRegions        string
	IncidentDocTemplateURL string
}

func FromViper(v *viper.Viper) (Config, error) {
	c := Config{}

	c.NS = v.GetString("server.prometheusNamespace")

	c.SlackAccessToken = v.GetString("slack.accessToken")
	c.SlackSigningSecret = v.GetString("slack.signingSecret")
	c.SlackAdminGroupID = v.GetString("slack.adminGroupID")
	c.SlackChannelID = v.GetString("slack.channelID")

	c.Addr = v.GetString("addr")
	c.TLSAddr = v.GetString("tls.addr")
	c.TLSCert = v.GetString("tls.cert")
	c.TLSKey = v.GetString("tls.key")

	c.IncidentEnvs = v.GetString("incident.environments")
	c.IncidentRegions = v.GetString("incident.regions")
	c.IncidentDocTemplateURL = v.GetString("incidentDocTemplateURL")

	return c, nil
}
