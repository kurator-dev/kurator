package generic

import (
	"io/ioutil"

	"github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"
)

func (g *Options) FlagSet(n string) *flag.FlagSet {
	f := flag.NewFlagSet(n, flag.ExitOnError)
	f.SetOutput(ioutil.Discard)

	// Set the default Usage to empty
	f.Usage = func() {
		logrus.Errorf(f.FlagUsages())
	}

	return f
}
