/*
Copyright 2022-2025 Kurator Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

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
