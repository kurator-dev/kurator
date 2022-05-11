package command

import (
	"encoding/json"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/zirain/ubrain/pkg/generic"

	"github.com/zirain/ubrain/pkg/version"
)

type VersionCommand struct {
	*generic.Options
}

func (c *VersionCommand) Run(args []string) int {
	v := version.Get()

	y, err := json.MarshalIndent(&v, "", "  ")
	if err != nil {
		logrus.Errorf("Error marshal version: %s", err.Error())
		return 1
	}

	c.Ui.Output(string(y))

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
