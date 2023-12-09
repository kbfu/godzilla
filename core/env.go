package core

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var (
	LocalDebug            = populateEnv("LOCAL_DEBUG", false).(bool)
	LogHouse              = populateEnv("LOG_HOUSE", "github-k8s-runner").(string)
	GithubWorkerName      = populateEnv("ACTIONS_RUNNER_POD_NAME", "").(string)
	GithubWorkDir         = populateEnv("GITHUB_WORKSPACE", "").(string)
	GithubWorkerNamespace = populateEnv("ACTIONS_RUNNER_NAMESPACE", "cicd").(string)
	Scenario              = populateEnv("SCENARIO", "").(string)
	ChaosGitAddress       = populateEnv("CHAOS_GIT_ADDRESS", "").(string)
	ChaosGitRepo          = populateEnv("CHAOS_GIT_REPO", "").(string)
	ChaosGitBranch        = populateEnv("CHAOS_GIT_BRANCH", "main").(string)
	MysqlUser             = populateEnv("MYSQL_USER", "string").(string)
	MysqlPassword         = populateEnv("MYSQL_PASSWORD", "root").(string)
	MysqlHost             = populateEnv("MYSQL_HOST", "").(string)
	MysqlPort             = populateEnv("MYSQL_PORT", "").(string)
	MysqlDatabase         = populateEnv("MYSQL_DATABASE", "godzilla").(string)
)

func populateEnv(name string, defaultValue any) any {
	if name == "LOCAL_DEBUG" {
		if os.Getenv(name) != "" {
			val, err := strconv.ParseBool(os.Getenv("LOCAL_DEBUG"))
			if err != nil {
				logrus.Fatalf("Parse LOCAL_DEBUG error, reason, %s", err.Error())
			}
			return val
		}
	} else {
		if os.Getenv(name) != "" {
			return os.Getenv(name)
		}
	}

	return defaultValue
}

func ParseVars() {
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
