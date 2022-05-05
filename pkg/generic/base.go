package generic

import (
	"fmt"
	"io/ioutil"

	flag "github.com/spf13/pflag"
)

func (g *Options) FlagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ExitOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {
		g.Errorf(f.FlagUsages())
	}

	return f
}

func (g *Options) Infof(format string, a ...interface{}) {
	if g.Ui == nil {
		return
	}
	g.Ui.Output(fmt.Sprintf(format, a...))
}

func (g *Options) Errorf(format string, a ...interface{}) {
	if g.Ui == nil {
		return
	}
	g.Ui.Error(fmt.Sprintf(format, a...))
}
