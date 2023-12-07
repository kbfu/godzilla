package kube

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"godzilla/core"
	"godzilla/types"
	"gopkg.in/yaml.v3"
	"io"
	"io/fs"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"strings"
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

func (chaosJob *ChaosJob) Run() (actualJobName string, err error) {
	var job batchV1.Job
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		job = chaosJob.LitmusJob()
		actualJobName = job.Name
	}

	_, err = client.BatchV1().Jobs(chaosJob.Namespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
	return
}

func (chaosJob *ChaosJob) preCheck() bool {
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		return true
	}
	return false
}

func (chaosJob *ChaosJob) cleanJob(actualName string) error {
	policy := metaV1.DeletePropagationForeground
	return client.BatchV1().Jobs(chaosJob.Namespace).Delete(context.TODO(), actualName, metaV1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

func CreateChaos() error {
	// run all inside scenarios
	err := filepath.Walk("scenarios", func(path string, info fs.FileInfo, err error) error {
		scenarioName := strings.Split(info.Name(), ".")[0]
		if core.Scenario != "" && scenarioName != core.Scenario {
			return nil
		}
		var (
			chaosJobs [][]ChaosJob
			wg        sync.WaitGroup
		)
		if !info.IsDir() {
			logrus.Infof("running scenario: %s", scenarioName)
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(data, &chaosJobs)
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
						actualName, err := j.Run()
						if err != nil {
							logrus.Errorf("Job %s run failed, reason: %s", j.Name, err.Error())
						}
						// collect logs to files
						j.fetchChaosLogs(actualName)
						// cleanup
						err = j.cleanJob(actualName)
						if err != nil {
							logrus.Errorf("Job %s cleanup failed, reason: %s", j.Name, err.Error())
						}
						wg.Done()
					}()
				}
				wg.Wait()
			}
		}
		if core.Scenario != "" && scenarioName == core.Scenario {
			return io.EOF
		}
		return nil
	})
	if err != nil {
		return err
	}
	// deal with logs
	saveLogs()
	return nil
}

func saveLogs() {
	switch core.LogHouse {
	case "github-k8s-runner":
		filepath.Walk("logs", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				copyIntoPod(core.GithubWorkerName, core.GithubWorkerNamespace, path,
					fmt.Sprintf("%s/%s", core.GithubWorkDir, info.Name()))
			}
			return nil
		})
	}
}
