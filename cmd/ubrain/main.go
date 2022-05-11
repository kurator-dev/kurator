package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/mitchellh/cli"
	"github.com/sirupsen/logrus"
	"istio.io/pkg/env"

	"github.com/zirain/ubrain/pkg/command"
	"github.com/zirain/ubrain/pkg/generic"
)

func main() {
	initLogging()
	commands := initCommandFactory()
	runner := initRunner(commands)

	code, err := runner.Run()
	if err != nil {
		logrus.Errorf("CLI executing error: %s", err.Error())
	}
	os.Exit(code)
}

func initLogging() {
	levelEnv := env.RegisterStringVar("LOGGING_LEVEL", "info", "output logging level, Possible values: panic, fatal, error, warn, info, debug, trace").Get()
	level, err := logrus.ParseLevel(strings.ToLower(levelEnv))
	if err != nil {
		logrus.Errorf("parse logging level, use info level")
		level = logrus.InfoLevel
	}

	logrus.SetFormatter(&logrus.TextFormatter{
		TimestampFormat: "2006-01-02 15:04:05",
		FullTimestamp:   true,
	})
	logrus.SetLevel(level)
}

func initCommandFactory() map[string]cli.CommandFactory {
	o := generic.New()

	return map[string]cli.CommandFactory{
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Options: o,
			}, nil
		},
		"install istio": func() (cli.Command, error) {
			return &command.IstioInstallCommand{
				Options: o,
			}, nil
		},
		"install karmada": func() (cli.Command, error) {
			return &command.KarmadaInstallCommand{
				Options: o,
			}, nil
		},
		"install volcano": func() (cli.Command, error) {
			return &command.VolcanoInstallCommand{
				Options: o,
			}, nil
		},
		"join": func() (cli.Command, error) {
			return &command.JoinCommand{
				Options: o,
			}, nil
		},
	}
}

func initRunner(f map[string]cli.CommandFactory) *cli.CLI {
	binName := filepath.Base(os.Args[0])
	args := os.Args[1:]

	// Rebuild the CLI with any modified args.
	logrus.Debugf("CLI command args: %#v", args)
	return &cli.CLI{
		Name:       binName,
		Args:       args,
		Commands:   f,
		HelpWriter: os.Stdout,
	}
}
