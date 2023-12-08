package core

import (
	"os/exec"

	"github.com/sirupsen/logrus"
)

func CloneChaosRepo() {
	cmd := exec.Command("git", "clone", "-b", ChaosGitBranch, ChaosGitAddress)
	out, err := cmd.CombinedOutput()
	if err != nil {
		logrus.Fatal(string(out))
	}
	logrus.Info(string(out))
}
