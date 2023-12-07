package core

import (
	"github.com/sirupsen/logrus"
	"os/exec"
)

func CloneChaosRepo() {
	cmd := exec.Command("git", "clone", ChaosGitAddress)
	stdout, err := cmd.Output()
	if err != nil {
		logrus.Fatal(err)
	}
	logrus.Info(stdout)
}
