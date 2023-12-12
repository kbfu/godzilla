package chaos

import (
	"context"
	"errors"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/sirupsen/logrus"
	"godzilla/chaos/litmus/pod"
	"godzilla/db"
	"godzilla/env"
	"godzilla/types"
	"gopkg.in/yaml.v3"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"sync"
)

type ChaosJob struct {
	Name               string            `yaml:"name"`
	Type               string            `yaml:"type"`
	Config             map[string]string `yaml:"config"`
	Image              string            `yaml:"image"`
	ServiceAccountName string            `yaml:"serviceAccountName"`
	Status             JobStatus         `yaml:"status"`
	FailedReason       string            `yaml:"failedReason"`
}

func (chaosJob *ChaosJob) Run(jobStatusId uint) {
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		runLitmusCommon(chaosJob, jobStatusId)
	case string(types.LitmusPodIoStress):
		runLitmusStress(chaosJob, jobStatusId)
	}
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
			case string(types.LitmusPodDelete), string(types.LitmusPodIoStress):
			default:
				return errors.New(fmt.Sprintf("unsupported type %s", j.Type))
			}
		}
	}
	return nil
}

func (chaosJob *ChaosJob) cleanJob(jobStatusId uint) error {
	logrus.Infof("cleaning up the chaos job, status id: %v", jobStatusId)
	policy := metaV1.DeletePropagationForeground
	// get name
	jobList, err := client.BatchV1().Jobs(env.JobNamespace).List(context.TODO(), metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("chaos.job.id=%v", jobStatusId),
	})
	if err != nil {
		return err
	}
	for _, j := range jobList.Items {
		err := client.BatchV1().Jobs(env.JobNamespace).Delete(context.TODO(), j.Name, metaV1.DeleteOptions{
			PropagationPolicy: &policy,
		})
		if err != nil {
			return err
		}
	}
	return nil
}

type ChaosBody struct {
	Scenario         string            `json:"scenario" binding:"required"`
	OverriddenConfig map[string]string `json:"overriddenConfig,omitempty"`
}

func overrideConfig(chaosJobs [][]ChaosJob, body ChaosBody) {
	for i := range chaosJobs {
		for j := range chaosJobs[i] {
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
	logrus.Infof("override the configuration for %s", body.Scenario)
	overrideConfig(chaosJobs, body)

	// pre-check before run
	logrus.Infof("precheck for %s", body.Scenario)
	err = preCheck(chaosJobs)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError,
			ErrorResponse(InvalidScenario, err))
		return
	}

	jobStatusId, err := initStatus(chaosJobs, s.Id)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse(MySqlSaveError, err))
		return
	}
	logrus.Infof("scenario %s is ready now, current run id is %v", body.Scenario, jobStatusId)

	go func() {
		for _, parallelJobs := range chaosJobs {
			for _, j := range parallelJobs {
				wg.Add(1)
				j := j
				go func() {
					logrus.Infof("running scenario %s, id: %v, job %s", body.Scenario, jobStatusId, j.Name)
					j.Run(jobStatusId)
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}()
	c.JSON(http.StatusCreated, NormalResponse(Ok, ""))
}

func GetChaos(c *gin.Context) {
	scenario := c.Query("scenario")
	s := db.Scenario{Name: scenario}
	err := s.GetByName()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, ErrorResponse(MySqlError, err))
		return
	}
	if s.Definition == "" {
		c.AbortWithStatusJSON(http.StatusNotFound, ErrorResponse(ScenarioNotFound, nil))
		return
	}
	c.JSON(http.StatusOK, NormalResponse(Ok, s.Definition))
}
