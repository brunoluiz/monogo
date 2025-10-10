package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/alecthomas/kong"
)

type Context struct {
	context.Context
	Logger *slog.Logger
}

var cli struct {
	LogLevel slog.Level `help:"Log level to use for the application." default:"INFO" enum:"DEBUG,INFO,WARN,ERROR"`
	Detect   DetectCmd  `cmd:"" help:"Detect changed Golang packages based on git changes"`
	Version  VersionCmd `cmd:"" help:"Return version details"`
}

func main() {
	signalCtx, signalStop := signal.NotifyContext(context.Background(),
		syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT, os.Interrupt)
	defer signalStop()

	app := kong.Parse(&cli)
	logger := slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{
		Level: cli.LogLevel,
	}))

	err := app.Run(&Context{
		Context: signalCtx,
		Logger:  logger,
	})
	app.FatalIfErrorf(err)
}
