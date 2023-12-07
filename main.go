package main

import (
	"github.com/sirupsen/logrus"
	"godzilla/core"
	"godzilla/kube"
	"io"
)

func init() {
	core.InitLogrus()
	core.ParseVars()
	core.CloneChaosRepo()
	kube.InitKubeClient()
	//kube.ReadyChaosEnv()
}

func main() {
	err := kube.CreateChaos()
	if err != nil && err != io.EOF {
		logrus.Fatalf("Create chaos failed, reason %s", err.Error())
	}
}
