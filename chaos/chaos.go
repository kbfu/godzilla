package chaos

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"godzilla/db"
	"godzilla/env"
	"godzilla/types"
	"gopkg.in/yaml.v3"
	"io/fs"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"path/filepath"
	"reflect"
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

func (chaosJob *ChaosJob) Run(scenarioName string, overriddenConfig map[string]string) (actualJobName string, err error) {
	var job batchV1.Job
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		job = chaosJob.LitmusJob(scenarioName, overriddenConfig)
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
	logrus.Infof("Cleaning up the chaos job %s", actualName)
	policy := metaV1.DeletePropagationForeground
	return client.BatchV1().Jobs(chaosJob.Namespace).Delete(context.TODO(), actualName, metaV1.DeleteOptions{
		PropagationPolicy: &policy,
	})
}

type ChaosBody struct {
	Scenario         string            `json:"scenario" binding:"required"`
	OverriddenConfig map[string]string `json:"overriddenConfig,omitempty"`
}

func CreateChaos(c *gin.Context) {
	var body ChaosBody
	err := c.BindJSON(&body)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, ErrorResponse(RequestError, err))
		return
	}

	// run all inside scenarios
	var (
		chaosJobs [][]ChaosJob
		wg        sync.WaitGroup
	)
	logrus.Infof("getting scenario deifition for %s", body.Scenario)
	s := db.Scenario{Name: body.Scenario}
	err = s.GetByName()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse(ReadFileError, err))
		return
	}

	logrus.Infof("running scenario: %s", body.Scenario)
	data := s.Definition
	err = yaml.Unmarshal([]byte(data), &chaosJobs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse(YamlMarshalError, err))
		return
	}
	// pre-check before run
	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			if !j.preCheck() {
				c.AbortWithStatusJSON(http.StatusInternalServerError,
					ErrorResponse(InvalidScenario, errors.New(fmt.Sprintf("pre check failed, unknown type %s", j.Type))))
				return
			}
		}
	}

	go func() {
		for _, parallelJobs := range chaosJobs {
			for _, j := range parallelJobs {
				wg.Add(1)
				j := j
				go func() {
					actualName, err := j.Run(body.Scenario, body.OverriddenConfig)
					if err != nil {
						logrus.Errorf("Job %s run failed, reason: %s", j.Name, err.Error())
						// todo job status failed
						// todo scenario status failed
						return
					}
					// todo job status started
					// watch for the status
					pod := client.CoreV1().Pods(j.Namespace)
					w, err := pod.Watch(context.TODO(), metaV1.ListOptions{
						LabelSelector: "chaos.job=true",
					})
					if err != nil {
						logrus.Errorf("job %s status watch failed, reason: %s", j.Name, err.Error())
						// todo job status failed
						// todo scenario status failed
					}
					for c := range w.ResultChan() {
						if c.Object != nil {
							if reflect.ValueOf(c.Object).Type().Elem().Name() == "Pod" {
								podObject := c.Object.(*coreV1.Pod)
								if podObject.Status.Phase == coreV1.PodSucceeded || podObject.Status.Phase == coreV1.PodFailed {
									// cleanup
									err = j.cleanJob(actualName)
									if err != nil {
										logrus.Errorf("job %s cleanup failed, reason: %s", j.Name, err.Error())
										// todo job status failed
										// todo scenario status failed
									}
									break
								}
							}
						}
					}
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}()
	c.JSON(http.StatusCreated, NormalResponse(Ok, ""))
}

func saveLogs() {
	switch env.LogHouse {
	case "github-k8s-runner":
		filepath.Walk("logs", func(path string, info fs.FileInfo, err error) error {
			if !info.IsDir() {
				copyIntoPod(env.GithubWorkerName, env.GithubWorkerNamespace, "runner", path,
					fmt.Sprintf("%s/%s", env.GithubWorkDir, info.Name()))
			}
			return nil
		})
	}
}
