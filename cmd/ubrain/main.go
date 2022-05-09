package main

import (
	"log"
	"os"
	"path/filepath"

	"github.com/mitchellh/cli"

	"github.com/zirain/ubrain/pkg/command"
	"github.com/zirain/ubrain/pkg/generic"
)

func main() {
	commands := initCommandFactory()
	runner := initRunner(commands)

	code, err := runner.Run()
	if err != nil {
		log.Printf("[Error] CLI executing error: %s", err.Error())
	}
	os.Exit(code)
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
	}
}

func initRunner(f map[string]cli.CommandFactory) *cli.CLI {
	binName := filepath.Base(os.Args[0])
	args := os.Args[1:]

	// Rebuild the CLI with any modified args.
	log.Printf("CLI command args: %#v\n", args)
	return &cli.CLI{
		Name:       binName,
		Args:       args,
		Commands:   f,
		HelpWriter: os.Stdout,
	}
}
