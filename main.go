package main

import (
	"github.com/sirupsen/logrus"
	"godzilla/core"
	"godzilla/kube"
)

func init() {
	core.InitLogrus()
	core.ParseVars()
	kube.ReadyChaosEnv()
}

func main() {
	err := kube.CreateChaos()
	if err != nil {
		logrus.Fatalf("Create chaos failed, reason %s", err.Error())
	}
}
