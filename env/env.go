package env

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var (
	JobNamespace          = populateEnv("JOB_NAMESPACE", "test-chaos").(string)
	LocalDebug            = populateEnv("LOCAL_DEBUG", false).(bool)
	LogHouse              = populateEnv("LOG_HOUSE", "github-k8s-runner").(string)
	GithubWorkerName      = populateEnv("ACTIONS_RUNNER_POD_NAME", "").(string)
	GithubWorkDir         = populateEnv("GITHUB_WORKSPACE", "").(string)
	GithubWorkerNamespace = populateEnv("ACTIONS_RUNNER_NAMESPACE", "cicd").(string)
	MysqlUser             = populateEnv("MYSQL_USER", "root").(string)
	MysqlPassword         = populateEnv("MYSQL_PASSWORD", "root").(string)
	MysqlHost             = populateEnv("MYSQL_HOST", "127.0.0.1").(string)
	MysqlPort             = populateEnv("MYSQL_PORT", "3306").(string)
	MysqlDatabase         = populateEnv("MYSQL_DATABASE", "godzilla").(string)
)

func populateEnv(name string, defaultValue any) any {
	if name == "LOCAL_DEBUG" {
		if os.Getenv(name) != "" {
			val, err := strconv.ParseBool(os.Getenv("LOCAL_DEBUG"))
			if err != nil {
				logrus.Fatalf("parse LOCAL_DEBUG error, reason, %s", err.Error())
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
}
