package command

import (
	"fmt"
	"io/ioutil"

	flag "github.com/spf13/pflag"

	"github.com/zirain/ubrain/pkg/cli"
)

type Base struct {
	Settings *cli.Settings
}

func (b *Base) FlagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ContinueOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {}

	return f
}

func (b *Base) Infof(format string, a ...interface{}) {
	if b.Settings.Ui == nil {
		return
	}
	b.Settings.Ui.Output(fmt.Sprintf(format, a...))
}

func (b *Base) Errorf(format string, a ...interface{}) {
	if b.Settings.Ui == nil {
		return
	}
	b.Settings.Ui.Error(fmt.Sprintf(format, a...))
}
