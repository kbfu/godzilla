package chaos

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"godzilla/chaos/litmus/pod"
	"godzilla/db"
	"godzilla/types"
	"gopkg.in/yaml.v3"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
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
	Status             JobStatus         `yaml:"status"`
	FailedReason       string            `yaml:"failedReason"`
}

func (chaosJob *ChaosJob) Run(jobStatusId uint) (actualJobName string, err error) {
	var job batchV1.Job
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		job = chaosJob.LitmusJob(jobStatusId)
		actualJobName = job.Name
	}

	_, err = client.BatchV1().Jobs(chaosJob.Namespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
	return
}

func preCheck(chaosJobs [][]ChaosJob) error {
	dup := make(map[string]string)
	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			_, ok := dup[j.Name]
			if !ok {
				dup[j.Name] = ""
			} else {
				return errors.New(fmt.Sprintf("duplicate step name found: %s", j.Name))
			}
			switch j.Type {
			case string(types.LitmusPodDelete):
			default:
				return errors.New(fmt.Sprintf("unsupported type %s", j.Type))
			}
		}
	}
	return nil
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

func overrideConfig(chaosJobs [][]ChaosJob, body ChaosBody) {
	for i := range chaosJobs {
		for j := range chaosJobs[i] {
			switch chaosJobs[i][j].Type {
			case string(types.LitmusPodDelete):
				config := pod.PopulateDefaultDeletePod()
				// override default config
				for k, v := range config.Env {
					_, ok := chaosJobs[i][j].Config[k]
					if !ok {
						chaosJobs[i][j].Config[k] = v
					}
				}
				// allow it to be overridden by request body
				for k := range chaosJobs[i][j].Config {
					_, ok := body.OverriddenConfig[fmt.Sprintf("%s-%s", chaosJobs[i][j].Name, k)]
					if ok {
						chaosJobs[i][j].Config[k] = body.OverriddenConfig[fmt.Sprintf("%s-%s", chaosJobs[i][j].Name, k)]
					}
				}

				if chaosJobs[i][j].Image == "" {
					chaosJobs[i][j].Image = config.Image
				}
				if chaosJobs[i][j].ServiceAccountName == "" {
					chaosJobs[i][j].ServiceAccountName = config.ServiceAccountName
				}
			}
			chaosJobs[i][j].Status = PendingStatus
		}
	}
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
	// override the configuration
	overrideConfig(chaosJobs, body)

	jobStatusId, err := initStatus(chaosJobs, s.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse(MySqlSaveError, err))
		return
	}

	// pre-check before run
	err = preCheck(chaosJobs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			ErrorResponse(InvalidScenario, err))
		return
	}

	go func() {
		for _, parallelJobs := range chaosJobs {
			for _, j := range parallelJobs {
				wg.Add(1)
				j := j
				go func() {
					actualName, err := j.Run(jobStatusId)
					if err != nil {
						logrus.Errorf("Job %s run failed, reason: %s", j.Name, err.Error())
						j.Status = FailedStatus
						j.FailedReason = err.Error()
						statusChan <- map[uint]ChaosJob{jobStatusId: j}
						return
					}
					// job status started
					j.Status = RunningStatus
					statusChan <- map[uint]ChaosJob{jobStatusId: j}
					// watch for the status
					kubePod := client.CoreV1().Pods(j.Namespace)
					w, err := kubePod.Watch(context.TODO(), metaV1.ListOptions{
						LabelSelector: "chaos.job=true",
					})
					if err != nil {
						logrus.Errorf("job %s status watch failed, reason: %s", j.Name, err.Error())
						j.Status = FailedStatus
						j.FailedReason = err.Error()
						statusChan <- map[uint]ChaosJob{jobStatusId: j}
						return
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
										j.Status = FailedStatus
										j.FailedReason = err.Error()
										statusChan <- map[uint]ChaosJob{jobStatusId: j}
										return
									}
									switch podObject.Status.Phase {
									case coreV1.PodSucceeded:
										j.Status = SuccessStatus
									case coreV1.PodFailed:
										j.Status = FailedStatus
										j.FailedReason = "chaos job pod running failed"
									default:
										j.Status = UnknownStatus
									}
									statusChan <- map[uint]ChaosJob{jobStatusId: j}
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
