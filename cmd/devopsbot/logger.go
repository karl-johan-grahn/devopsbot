package main

import (
	"context"
	stdlog "log"
	"os"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/rs/zerolog/pkgerrors"
	"golang.org/x/term"
)

func initLogger(ctx context.Context) context.Context {
	zerolog.SetGlobalLevel(zerolog.InfoLevel)
	zerolog.DurationFieldInteger = false
	zerolog.DurationFieldUnit = time.Second
	zerolog.ErrorMarshalFunc = errorMarshalFunc
	zerolog.ErrorStackMarshaler = pkgerrors.MarshalStack

	stdlogger := log.With().Bool("stdlog", true).Logger()
	stdlog.SetFlags(0)
	stdlog.SetOutput(stdlogger)

	if term.IsTerminal(int(os.Stdout.Fd())) {
		// Use predefined layouts
		log.Logger = log.Output(zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: "15:04:05"})

		noLevelWriter := zerolog.ConsoleWriter{
			Out:         os.Stderr,
			FormatLevel: func(i interface{}) string { return "" },
		}
		stdlogger = stdlogger.Output(noLevelWriter)
		stdlog.SetOutput(stdlogger)
	}

	ctx = log.Logger.WithContext(ctx)
	return ctx
}

func errorMarshalFunc(err error) interface{} {
	return err
}
