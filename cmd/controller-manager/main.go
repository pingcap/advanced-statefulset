package main

import (
	"math/rand"
	"os"
	"time"

	"github.com/pingcap/advanced-statefulset/cmd/controller-manager/app"
	"k8s.io/component-base/logs"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	command := app.NewControllerManagerCommand()

	logs.InitLogs()
	defer logs.FlushLogs()

	if err := command.Execute(); err != nil {
		os.Exit(1)
	}
}
