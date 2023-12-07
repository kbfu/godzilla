package core

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var (
	LocalDebug            = false
	LogHouse              = "github-k8s-runner"
	GithubWorkerName      = ""
	GithubWorkDir         = ""
	GithubWorkerNamespace = "cicd"
)

func ParseVars() {
	var err error
	if os.Getenv("LOCAL_DEBUG") != "" {
		LocalDebug, err = strconv.ParseBool(os.Getenv("LOCAL_DEBUG"))
		if err != nil {
			logrus.Fatalf("Parse LOCAL_DEBUG error, reason, %s", err.Error())
		}
	}
	if os.Getenv("ACTIONS_RUNNER_POD_NAME") != "" {
		GithubWorkerName = os.Getenv("ACTIONS_RUNNER_POD_NAME")
	}
	if os.Getenv("GITHUB_WORKSPACE") != "" {
		GithubWorkDir = os.Getenv("GITHUB_WORKSPACE")
	}
}
