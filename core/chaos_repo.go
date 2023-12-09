package core

import (
	"github.com/sirupsen/logrus"
	"godzilla/db"
	"math/rand"
	"os/exec"
	"time"
)

func CloneChaosRepo() {
	repo := db.Repo{}
	rows, err := repo.FetchAll()
	if err != nil {
		logrus.Fatal(err)
	}
	for _, row := range rows {
		cmd := exec.Command("git", "clone", "-b", row.Branch, row.Address, row.Name)
		out, err := cmd.CombinedOutput()
		if err != nil {
			logrus.Fatal(string(out))
		}
		logrus.Info(string(out))
		row := row
		go func() {
			for {
				time.Sleep(time.Duration(rand.Intn(60)) * time.Second)
				cmd := exec.Command("git", "-C", row.Name, "pull")
				out, err := cmd.CombinedOutput()
				if err != nil {
					logrus.Errorf("update for repo %s failed, reason: %s", row.Name, string(out))
					continue
				}
			}
		}()
	}
}
