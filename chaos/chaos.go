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
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"sync"
	"time"
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
	var job batchV1.Job
	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		job = chaosJob.LitmusJob(jobStatusId)
		_, err := client.BatchV1().Jobs(env.JobNamespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
		if err != nil {
			logrus.Errorf("Job %s run failed, reason: %s", chaosJob.Name, err.Error())
			chaosJob.Status = FailedStatus
			chaosJob.FailedReason = err.Error()
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
			return
		}
		// job status started
		chaosJob.Status = RunningStatus
		statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
		// watch for the status
		kubePod := client.CoreV1().Pods(env.JobNamespace)
		w, err := kubePod.Watch(context.TODO(), metaV1.ListOptions{
			LabelSelector: fmt.Sprintf("chaos.job.id=%v,chaos.job.name=%s", jobStatusId, chaosJob.Name),
		})
		if err != nil {
			logrus.Errorf("job %s status watch failed, reason: %s", chaosJob.Name, err.Error())
			chaosJob.Status = FailedStatus
			chaosJob.FailedReason = err.Error()
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
			return
		}
		for c := range w.ResultChan() {
			if c.Object != nil {
				if reflect.ValueOf(c.Object).Type().Elem().Name() == "Pod" {
					podObject := c.Object.(*coreV1.Pod)
					if podObject.Status.Phase == coreV1.PodSucceeded || podObject.Status.Phase == coreV1.PodFailed {
						// cleanup
						err = chaosJob.cleanJob(jobStatusId)
						if err != nil {
							logrus.Errorf("job %s cleanup failed, reason: %s", chaosJob.Name, err.Error())
							chaosJob.Status = FailedStatus
							chaosJob.FailedReason = err.Error()
							statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
							return
						}
						switch podObject.Status.Phase {
						case coreV1.PodSucceeded:
							chaosJob.Status = SuccessStatus
						case coreV1.PodFailed:
							chaosJob.Status = FailedStatus
							chaosJob.FailedReason = "chaos job pod running failed"
						default:
							chaosJob.Status = UnknownStatus
						}
						statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
						break
					}
				}
			}
		}
	case string(types.LitmusPodIoStress):
		go func() {
			start := time.Now().Unix()
			duration, _ := strconv.Atoi(chaosJob.Config["TOTAL_CHAOS_DURATION"])
			elapsed := int(start) + duration
			var targetPods []string
			if chaosJob.Config["TARGET_PODS"] != "" {
				targetPods = strings.Split(chaosJob.Config["TARGET_PODS"], ",")
			}
			var percentage int
			if chaosJob.Config["PODS_AFFECTED_PERC"] == "" {
				percentage = 0
			} else {
				percentage, _ = strconv.Atoi(chaosJob.Config["PODS_AFFECTED_PERC"])
			}

			var pods []coreV1.Pod

			// if TARGET_PODS is not empty, use it
			if len(targetPods) > 0 {
				for _, targetPod := range targetPods {
					targetPod = strings.TrimSpace(targetPod)
					podObject, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).Get(context.TODO(), targetPod, metaV1.GetOptions{})
					if err != nil {
						chaosJob.Status = FailedStatus
						chaosJob.FailedReason = err.Error()
						statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
						return
					}
					pods = append(pods, *podObject)
				}
			} else {
				// check label
				podList, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).List(context.TODO(), metaV1.ListOptions{
					LabelSelector: chaosJob.Config["APP_LABEL"],
				})
				if err != nil {
					chaosJob.Status = FailedStatus
					chaosJob.FailedReason = err.Error()
					statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
					return
				}
				pods = podList.Items
			}
			// set the pods as percentage
			if percentage == 0 {
				pods = pods[:1]
			} else {
				pods = pods[:len(pods)*percentage/100]
			}

			for _, podObject := range pods {
				if podObject.Status.Phase == coreV1.PodRunning {
					// need to fetch the target node name
					nodeName := podObject.Spec.NodeName
					podName := podObject.Name
					chaosJob.Config["APP_POD"] = podName
					chaosJob.Config["CPU_CORES"] = "0"
					if chaosJob.Config["APP_CONTAINER"] == "" {
						chaosJob.Config["APP_CONTAINER"] = podObject.Spec.Containers[0].Name
					}
					chaosJob.Config["FILESYSTEM_UTILIZATION_PERCENTAGE"] = "0"
					chaosJob.Config["STRESS_TYPE"] = "pod-io-stress"
					job = chaosJob.LitmusJobStress(jobStatusId, nodeName, podName)
					_, err := client.BatchV1().Jobs(env.JobNamespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
					if err != nil {
						chaosJob.Status = FailedStatus
						chaosJob.FailedReason = err.Error()
						statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
						return
					}
				}
			}

			// job status started
			chaosJob.Status = RunningStatus
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}

			// only label pods needs to be watched
			for {
				targets, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).List(context.TODO(), metaV1.ListOptions{
					LabelSelector: chaosJob.Config["APP_LABEL"],
				})
				if err != nil {
					chaosJob.Status = FailedStatus
					chaosJob.FailedReason = err.Error()
					statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
					return
				}
				var chaosPods []coreV1.Pod
				runningChaosPods, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).List(context.TODO(), metaV1.ListOptions{
					LabelSelector: fmt.Sprintf("chaos.job.id=%v", jobStatusId),
					FieldSelector: "status.phase=Running",
				})
				if err != nil {
					chaosJob.Status = FailedStatus
					chaosJob.FailedReason = err.Error()
					statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
					return
				}
				pendingChaosPods, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).List(context.TODO(), metaV1.ListOptions{
					LabelSelector: fmt.Sprintf("chaos.job.id=%v", jobStatusId),
					FieldSelector: "status.phase=Pending",
				})
				if err != nil {
					chaosJob.Status = FailedStatus
					chaosJob.FailedReason = err.Error()
					statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
					return
				}
				chaosPods = append(chaosPods, runningChaosPods.Items...)
				chaosPods = append(chaosPods, pendingChaosPods.Items...)

				var pods []coreV1.Pod
				if len(chaosPods) == 0 {
					if percentage == 0 {
						pods = targets.Items[:1]
					} else {
						pods = targets.Items[:len(targets.Items)*percentage/100]
					}
				} else {
					// todo recalculate the pods
				}
				for _, podObject := range pods {
					if podObject.Status.Phase == coreV1.PodRunning {
						// need to fetch the target node name
						nodeName := podObject.Spec.NodeName
						podName := podObject.Name
						chaosJob.Config["APP_POD"] = podName
						chaosJob.Config["CPU_CORES"] = "0"
						if chaosJob.Config["APP_CONTAINER"] == "" {
							chaosJob.Config["APP_CONTAINER"] = podObject.Spec.Containers[0].Name
						}
						chaosJob.Config["TOTAL_CHAOS_DURATION"] = fmt.Sprintf("%v", elapsed-int(time.Now().Unix()))
						chaosJob.Config["FILESYSTEM_UTILIZATION_PERCENTAGE"] = "0"
						chaosJob.Config["STRESS_TYPE"] = "pod-io-stress"
						job = chaosJob.LitmusJobStress(jobStatusId, nodeName, podName)
						_, err := client.BatchV1().Jobs(env.JobNamespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
						if err != nil {
							chaosJob.Status = FailedStatus
							chaosJob.FailedReason = err.Error()
							statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
							return
						}
					}
				}

				// stop monitoring if time elapsed
				if elapsed < int(time.Now().Unix()) {
					// todo cleanup the chaos pods here
					break
				}
				time.Sleep(time.Second)
			}
			// check the chaos pods status now
		}()
	}
}

func filterPods(prev, curr []coreV1.Pod) []coreV1.Pod {
	var left []coreV1.Pod
	for i := range prev {
		for j := range curr {
			if prev[i].Name == curr[j].Name {
				left = append(left, prev[i])
			}
		}
	}
	return left
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
	overrideConfig(chaosJobs, body)

	// pre-check before run
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

	go func() {
		for _, parallelJobs := range chaosJobs {
			for _, j := range parallelJobs {
				wg.Add(1)
				j := j
				go func() {
					j.Run(jobStatusId)
					wg.Done()
				}()
			}
			wg.Wait()
		}
	}()
	c.JSON(http.StatusCreated, NormalResponse(Ok, ""))
}
