package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/urfave/cli/v2"
	"github.com/vshn/exoscale-metrics-collector/pkg/cmd"
	"github.com/vshn/exoscale-metrics-collector/pkg/log"
)

var (
	// these variables are populated by Goreleaser when releasing
	version = "unknown"
	commit  = "-dirty-"
	date    = time.Now().Format("2006-01-02")

	appName     = "metrics-collector"
	appLongName = "Metrics collector which gathers metrics information for cloud services"
)

func init() {
	// Remove `-v` short option from --version flag
	cli.VersionFlag.(*cli.BoolFlag).Aliases = nil
}

func main() {
	ctx, stop, app := newApp()
	defer stop()
	err := app.RunContext(ctx, os.Args)
	// If required flags aren't set, it will return with error before we could set up logging
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}

func newApp() (context.Context, context.CancelFunc, *cli.App) {
	var (
		logLevel  int
		logFormat string
	)
	app := &cli.App{
		Name:    appName,
		Usage:   appLongName,
		Version: fmt.Sprintf("%s, revision=%s, date=%s", version, commit, date),

		EnableBashCompletion: true,

		Flags: []cli.Flag{
			&cli.IntFlag{
				Name:        "log-level",
				Aliases:     []string{"v"},
				EnvVars:     []string{"LOG_LEVEL"},
				Usage:       "number of the log level verbosity",
				Value:       0,
				Destination: &logLevel,
			},
			&cli.StringFlag{
				Name:        "log-format",
				EnvVars:     []string{"LOG_FORMAT"},
				Usage:       "sets the log format (values: [json, console])",
				DefaultText: "console",
				Destination: &logFormat,
			},
		},
		Before: func(c *cli.Context) error {
			logger, err := log.NewLogger(appName, version, logLevel, logFormat)
			if err != nil {
				return fmt.Errorf("before: %w", err)
			}
			c.Context = log.NewLoggingContext(c.Context, logger)
			return nil
		},
		Action: func(c *cli.Context) error {
			if true {
				return cli.ShowAppHelp(c)
			}
			log.AppLogger(c.Context).WithValues(
				"date", date,
				"commit", commit,
				"go_os", runtime.GOOS,
				"go_arch", runtime.GOARCH,
				"go_version", runtime.Version(),
				"uid", os.Getuid(),
				"gid", os.Getgid(),
			).Info("Starting up " + appName)
			return nil
		},
		Commands: []*cli.Command{
			cmd.ExoscaleCmds(),
			cmd.CloudscaleCmds(),
		},
		ExitErrHandler: func(c *cli.Context, err error) {
			if err != nil {
				log.AppLogger(c.Context).Error(err, "fatal error")
				cli.HandleExitCoder(cli.Exit("", 1))
			}
		},
	}

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	return ctx, stop, app
}
