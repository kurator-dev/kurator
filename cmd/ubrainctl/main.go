package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/mitchellh/cli"

	ubraincli "github.com/zirain/ubrain/pkg/cli"
	"github.com/zirain/ubrain/pkg/command"
)

// Commands is the mapping of all the available commands.
var Commands map[string]cli.CommandFactory

func main() {
	initCommands()
	os.Exit(Run())
}

func Run() int {
	binName := filepath.Base(os.Args[0])
	args := os.Args[1:]

	// Rebuild the CLI with any modified args.
	fmt.Printf("CLI command args: %#v\n", args)
	cliRunner := &cli.CLI{
		Name:       binName,
		Args:       args,
		Commands:   Commands,
		HelpWriter: os.Stdout,
	}

	exitCode, err := cliRunner.Run()
	if err != nil {
		log.Printf("[Error] CLI executing error: %s", err.Error())
		return 1
	}

	return exitCode
}

func initCommands() {
	s := ubraincli.New()
	b := command.Base{
		Settings: s,
	}

	Commands = map[string]cli.CommandFactory{
		"version": func() (cli.Command, error) {
			return &command.VersionCommand{
				Base: b,
			}, nil
		},
		"install istio": func() (cli.Command, error) {
			return &command.IstioInstallCommad{
				Base: b,
			}, nil
		},
	}
}
