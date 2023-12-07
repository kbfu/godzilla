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
	Scenario              = ""
	ChaosGitAddress       = ""
	ChaosGitRepo          = ""
	ChaosGitBranch        = "main"
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
	if os.Getenv("SCENARIO") != "" {
		Scenario = os.Getenv("SCENARIO")
	}
	if os.Getenv("LOG_HOUSE") != "" {
		LogHouse = os.Getenv("LOG_HOUSE")
	}
	if os.Getenv("ACTIONS_RUNNER_NAMESPACE") != "" {
		GithubWorkerNamespace = os.Getenv("ACTIONS_RUNNER_NAMESPACE")
	}
	if os.Getenv("CHAOS_GIT_ADDRESS") != "" {
		ChaosGitAddress = os.Getenv("CHAOS_GIT_ADDRESS")
	}
	if os.Getenv("CHAOS_GIT_REPO") != "" {
		ChaosGitRepo = os.Getenv("CHAOS_GIT_REPO")
	}
	if os.Getenv("CHAOS_GIT_BRANCH") != "" {
		ChaosGitBranch = os.Getenv("CHAOS_GIT_BRANCH")
	}

	logrus.Info("vars for current run")
	logrus.Infof("LOCAL_DEBUG: %v", LocalDebug)
	logrus.Infof("LOG_HOUSE %s", LogHouse)
	logrus.Infof("ACTIONS_RUNNER_POD_NAME %s", GithubWorkerName)
	logrus.Infof("GITHUB_WORKSPACE %s", GithubWorkDir)
	logrus.Infof("ACTIONS_RUNNER_NAMESPACE %s", GithubWorkerNamespace)
	logrus.Infof("SCENARIO %s", Scenario)
	logrus.Infof("CHAOS_GIT_ADDRESS %s", ChaosGitAddress)
	logrus.Infof("CHAOS_GIT_REPO %s", ChaosGitRepo)
	logrus.Infof("CHAOS_GIT_BRANCH %s", ChaosGitBranch)
}
