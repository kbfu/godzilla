package main

import (
	"github.com/sirupsen/logrus"
	"godzilla/chaos"
	"godzilla/core"
	"godzilla/db"
	"godzilla/env"
)

func init() {
	core.InitLogrus()
	env.ParseVars()
	db.Open()
	chaos.InitKubeClient()
	go chaos.StatusWorker()
	//kube.ReadyChaosEnv()
}

func main() {
	r := core.SetupRouter()

	err := r.Run()
	if err != nil {
		logrus.Fatal(err)
	}
}
