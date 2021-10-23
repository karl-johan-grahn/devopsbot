package main

import (
	"context"
	"net/http"
	"net/http/httputil"
	"os"
	"os/signal"
	"time"

	"github.com/gorilla/handlers"
	devopsbot "github.com/karl-johan-grahn/devopsbot"
	"github.com/karl-johan-grahn/devopsbot/bot"
	"github.com/karl-johan-grahn/devopsbot/config"
	"github.com/karl-johan-grahn/devopsbot/internal/middleware"
	"github.com/karl-johan-grahn/devopsbot/version"
	"github.com/slack-go/slack"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/hlog"
	"github.com/rs/zerolog/log"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

const (
	tlsAddr            = "tls.addr"
	tlsCert            = "tls.cert"
	tlsKey             = "tls.key"
	slackAccessToken   = "slack.accessToken"
	slackSigningSecret = "slack.signingSecret"
	slackAdminGroup    = "slack.adminGroupID"
	broadcastChannelID = "slack.broadcastChannelID"
)

func initFlags(cmd *cobra.Command) {
	cmd.Flags().SortFlags = false

	cmd.Flags().StringP("addr", "a", ":3333", "address:port to listen on")

	cmd.Flags().String(tlsAddr, ":3443", "address:port to listen on for TLS")
	cmd.Flags().String(tlsCert, "devopsbot.pem", "Path to TLS certificate")
	cmd.Flags().String(tlsKey, "devopsbot-key.pem", "Path to TLS private key")

	cmd.Flags().BoolP("verbose", "v", false, "Output extra logs")
	cmd.Flags().BoolP("trace", "t", false, "Output trace logs")

	cmd.Flags().String(slackAccessToken, "", "Slack bot access token")
	cmd.Flags().String(slackSigningSecret, "", "Slack bot signing secret")
	cmd.Flags().String(slackAdminGroup, "", "Slack ID for the admin user group")
	cmd.Flags().String(broadcastChannelID, "", "Slack ID for the channel to use as the broadcast channel")

	cmd.Flags().String("config", "config.yaml", "Config file to read (optional)")

	// Disable false positive lint
	//nolint:errcheck
	viper.BindPFlags(cmd.Flags())

	cobra.OnInitialize(initConfig(cmd))
}

func initConfig(cmd *cobra.Command) func() {
	return func() {
		configRequired := false
		if cmd.Flags().Changed("config") {
			// Use config file from the flag if it's explicitly set
			viper.SetConfigFile(viper.GetString("config"))
			configRequired = true
		} else {
			viper.AddConfigPath(".")
			viper.AddConfigPath("./config")
			viper.AddConfigPath("../config")

			viper.SetConfigName("config")
		}

		err := viper.ReadInConfig()
		if err != nil && configRequired {
			log.Fatal().Msgf("Specified config file '%s' missing: %v", viper.GetString("config"), err)
		}

		_ = viper.BindEnv("verbose", "verbose")
		_ = viper.BindEnv("trace", "trace")
		if viper.GetBool("verbose") {
			zerolog.SetGlobalLevel(zerolog.DebugLevel)
		}

		if viper.GetBool("trace") {
			zerolog.SetGlobalLevel(zerolog.TraceLevel)
		}

		if err != nil {
			log.Trace().Msgf("Specified config file '%s' missing: %v", viper.GetString("config"), err)
		} else {
			log.Debug().Msgf("Using config file: %s", viper.ConfigFileUsed())
		}

		_ = viper.BindEnv("server.prometheusNamespace", "server.prometheusNamespace")
		_ = viper.BindEnv("incidentDocTemplateURL", "incidentDocTemplateURL")
		_ = viper.BindEnv("incident.environments", "incident.environments")
		_ = viper.BindEnv("incident.regions", "incident.regions")
		_ = viper.BindEnv("addr", "addr")
		_ = viper.BindEnv(tlsAddr, tlsAddr)
		_ = viper.BindEnv(tlsCert, tlsCert)
		_ = viper.BindEnv(tlsKey, tlsKey)
		_ = viper.BindEnv(slackAccessToken, slackAccessToken)
		_ = viper.BindEnv(slackSigningSecret, slackSigningSecret)
		_ = viper.BindEnv(slackAdminGroup, slackAdminGroup)
		_ = viper.BindEnv(broadcastChannelID, broadcastChannelID)
	}
}

func main() {
	ctx := context.Background()
	ctx = initLogger(ctx)

	command := newCmd()
	initFlags(command)
	if err := command.ExecuteContext(ctx); err != nil {
		log.Fatal().Err(err).Msg(command.Name() + " failed")
	}
}

func newCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "devopsbot",
		Short:   "devopsbot improves efficiency by automating tasks",
		Version: version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx, cancel := context.WithCancel(cmd.Context())
			log := zerolog.Ctx(ctx)
			defer cancel()

			// update the logger's own level
			*log = log.Level(zerolog.GlobalLevel())

			log.Info().
				Str("version", version.Version).
				Str("revision", version.Revision).
				Str("cmd", cmd.CalledAs()).
				Msg("starting bot")
			cmd.SilenceErrors = true
			cmd.SilenceUsage = true

			cfg, err := config.FromViper(viper.GetViper())
			if err != nil {
				return err
			}

			slackClient := slack.New(cfg.SlackAccessToken,
				slack.OptionDebug(viper.GetBool("verbose")),
				slack.OptionHTTPClient(&http.Client{Transport: &spyTransport{rt: http.DefaultTransport}}),
			)
			opts := bot.Opts{
				SigningSecret:          cfg.SlackSigningSecret,
				AdminGroupID:           cfg.SlackAdminGroupID,
				BroadcastChannelID:     cfg.BroadcastChannelID,
				IncidentDocTemplateURL: cfg.IncidentDocTemplateURL,
				IncidentEnvs:           cfg.IncidentEnvs,
				IncidentRegions:        cfg.IncidentRegions,
			}
			log.Debug().Msgf("opts: %#v", opts)

			mux := http.NewServeMux()
			mux.Handle("/", devopsbot.HealthHandler(cfg.NS))
			mux.Handle("/bot/", http.StripPrefix("/bot", bot.NewBot(slackClient, opts)))

			h := http.Handler(mux)
			h = middleware.Logger(ctx, handlers.CompressHandler(h))

			// Run the servers non-blocking in goroutines so we can shut down gracefully
			httpSrv := &http.Server{Addr: cfg.Addr, Handler: h}
			go func() {
				log.Info().Str("addr", cfg.Addr).Msg("listening on HTTP")
				if err := httpSrv.ListenAndServe(); err != nil {
					log.Error().Err(err).Send()
				}
			}()

			httpsSrv := &http.Server{Addr: cfg.TLSAddr, Handler: h}
			go func() {
				log.Info().Str("addr", cfg.TLSAddr).Msg("listening on HTTPS")
				if err := httpsSrv.ListenAndServeTLS(cfg.TLSCert, cfg.TLSKey); err != nil {
					log.Error().Err(err).Send()
				}
			}()

			ch := make(chan os.Signal, 1)
			// Handle SIGINT (Ctrl+C)
			signal.Notify(ch, os.Interrupt)
			<-ch

			// Parent already defers cancel above, so disable false positive lint
			//nolint:govet
			ctx, _ = context.WithTimeout(ctx, 15*time.Second)

			log.Info().Msg("shutting down")

			err = httpSrv.Shutdown(ctx)
			if err != nil {
				return err
			}
			return httpsSrv.Shutdown(ctx)
		},
	}
	return cmd
}

type spyTransport struct {
	rt http.RoundTripper
}

func (t *spyTransport) RoundTrip(req *http.Request) (resp *http.Response, err error) {
	log := hlog.FromRequest(req)
	if log.GetLevel() == zerolog.TraceLevel {
		b, _ := httputil.DumpRequestOut(req, true)
		log.Trace().Bytes("request", b).Msg("dumping request")
	}
	resp, err = t.rt.RoundTrip(req)
	if resp != nil && log.GetLevel() == zerolog.TraceLevel {
		b, _ := httputil.DumpResponse(resp, true)
		log.Trace().Bytes("response", b).Msg("dumping response")
	}
	return resp, err
}
