package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/sirupsen/logrus"

	"istio.io/pkg/env"

	"kurator.dev/kurator/cmd/kurator/app"
)

func main() {
	initLogging()

	if err := app.Run(); err != nil {
		fmt.Println("execute kurator command failed: ", err)
		os.Exit(1)
	}
}

// TODO: move to pkg/bootstrap
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
