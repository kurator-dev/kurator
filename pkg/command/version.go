package command

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/zirain/ubrain/pkg/version"
)

type VersionCommand struct {
	Base
}

func (c *VersionCommand) Run(args []string) int {
	v := version.Get()

	y, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		c.Settings.Ui.Error(fmt.Sprintf("Error unmarshall version: %s\n", err.Error()))
		return 1
	}

	c.Settings.Ui.Output(string(y))

	return 0
}

func (c *VersionCommand) Help() string {
	helpText := `
Usage: ubrain version

  Displays the version of Ubrain

`
	return strings.TrimSpace(helpText)
}

func (c *VersionCommand) Synopsis() string {
	return "Show the current version"
}
