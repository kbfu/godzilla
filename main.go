package main

import (
	"godzilla/core"
	"godzilla/kube"
)

func init() {
	core.InitLogrus()
	core.ParseVars()
	kube.ReadyChaosEnv()
}

func main() {
	kube.CreateJob("")
}
