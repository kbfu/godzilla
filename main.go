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

	//err := kube.CreateChaos()
	//if err != nil && err != io.EOF {
	//	logrus.Fatalf("Create chaos failed, reason %s", err.Error())
	//}
}
