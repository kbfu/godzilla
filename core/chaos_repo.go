package core

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

func CloneChaosRepo() {
	cmd := exec.Command("git", "clone", "-b", ChaosGitBranch, ChaosGitAddress)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Info(string(out))
		logrus.Fatal(err)
	}
	logrus.Info(string(out))
}
