package kube

import (
	"context"
	"github.com/sirupsen/logrus"
	"godzilla/types"
	"gopkg.in/yaml.v3"
	"io/fs"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sync"
)

type ChaosJob struct {
	Name               string            `yaml:"name"`
	Type               string            `yaml:"type"`
	Namespace          string            `yaml:"namespace"`
	Config             map[string]string `yaml:"config"`
	Image              string            `yaml:"image"`
	ServiceAccountName string            `yaml:"serviceAccountName"`
}

func (chaosJob *ChaosJob) Run() error {
	var job batchV1.Job
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		job = chaosJob.LitmusJob()
	}

	_, err := client.BatchV1().Jobs(chaosJob.Namespace).Create(context.Background(), &job, metaV1.CreateOptions{})
	return err
}

func (chaosJob *ChaosJob) preCheck() bool {
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		return true
	}
	return false
}

func CreateChaos() error {
	var (
		chaosJobs [][]ChaosJob
		wg        sync.WaitGroup
	)
	err := filepath.Walk("scenarios", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(data, &chaosJobs)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	// precheck before run
	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			if !j.preCheck() {
				logrus.Fatalf("Pre check failed, unknown type %s", j.Type)
			}
		}
	}

	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			wg.Add(1)
			j := j
			go func() {
				err := j.Run()
				if err != nil {
					logrus.Fatal(err)
				}
				// todo need to watch chaos jobs here
				// todo collect logs to files
				// todo copy back to github actions worker if needed
				wg.Done()
			}()
		}
		wg.Wait()
	}
	return nil
}
