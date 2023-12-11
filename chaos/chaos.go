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
	"k8s.io/apimachinery/pkg/watch"
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
		logrus.Infof("creating job %s, run id %v", chaosJob.Name, jobStatusId)
		_, err := client.BatchV1().Jobs(env.JobNamespace).Create(context.TODO(), &job, metaV1.CreateOptions{})
		if err != nil {
			logrus.Errorf("job %s run failed, reason: %s", chaosJob.Name, err.Error())
			chaosJob.Status = FailedStatus
			chaosJob.FailedReason = err.Error()
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
			return
		}
		logrus.Infof("job %s created, run id %v", chaosJob.Name, jobStatusId)
		// job status started
		chaosJob.Status = RunningStatus
		statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
		// watch for the status
		w, err := client.CoreV1().Pods(env.JobNamespace).Watch(context.TODO(), metaV1.ListOptions{
			LabelSelector: fmt.Sprintf("chaos.job.id=%v,chaos.job.name=%s", jobStatusId, chaosJob.Name),
		})
		if err != nil {
			logrus.Errorf("job %s status watch failed, reason: %s", chaosJob.Name, err.Error())
			chaosJob.Status = FailedStatus
			chaosJob.FailedReason = err.Error()
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
			return
		}
		logrus.Infof("watching for job %s, run id %v", chaosJob.Name, jobStatusId)
		for event := range w.ResultChan() {
			if event.Object != nil {
				if reflect.ValueOf(event.Object).Type().Elem().Name() == "Pod" {
					podObject := event.Object.(*coreV1.Pod)
					if podObject.Status.Phase == coreV1.PodSucceeded || podObject.Status.Phase == coreV1.PodFailed {
						// cleanup
						logrus.Infof("job %s finished, run id %v, starting cleanup", chaosJob.Name, jobStatusId)
						err = chaosJob.cleanJob(jobStatusId)
						if err != nil {
							logrus.Errorf("job %s cleanup failed, reason: %s", chaosJob.Name, err.Error())
							chaosJob.Status = FailedStatus
							chaosJob.FailedReason = err.Error()
							statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
							w.Stop()
							return
						}
						logrus.Infof("job %s cleanup done, run id %v", chaosJob.Name, jobStatusId)
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
						w.Stop()
						break
					}
				}
			}
		}
	case string(types.LitmusPodIoStress):
		start := time.Now().Unix()
		duration, _ := strconv.Atoi(chaosJob.Config["TOTAL_CHAOS_DURATION"])
		elapsed := int(start) + duration
		go func() {
			logrus.Infof("creating job %s, run id %v", chaosJob.Name, jobStatusId)
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

			var (
				pods    []coreV1.Pod
				allPods []coreV1.Pod
			)

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
					FieldSelector: "status.phase=Running",
				})
				if err != nil {
					chaosJob.Status = FailedStatus
					chaosJob.FailedReason = err.Error()
					statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
					return
				}
				for i := range podList.Items {
					for _, condition := range podList.Items[i].Status.Conditions {
						if condition.Type == coreV1.ContainersReady && condition.Status == coreV1.ConditionTrue {
							if podList.Items[i].ObjectMeta.DeletionTimestamp == nil {
								allPods = append(allPods, podList.Items[i])
								pods = append(pods, podList.Items[i])
							}
						}
					}
				}
			}
			logrus.Infof("the total running pods is %d", len(allPods))
			// set the pods as percentage
			if percentage == 0 {
				pods = pods[:1]
			} else {
				pods = pods[:len(pods)*percentage/100]
			}
			var podNames []string
			for i := range pods {
				podNames = append(podNames, pods[i].Name)
			}
			logrus.Infof("the target pods are %v", podNames)

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
			logrus.Infof("job %s created, run id %v", chaosJob.Name, jobStatusId)

			// job status started
			chaosJob.Status = RunningStatus
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}

			// only label pods needs to be watched
			w, err := client.CoreV1().Pods(chaosJob.Config["APP_NAMESPACE"]).Watch(context.TODO(), metaV1.ListOptions{
				LabelSelector: chaosJob.Config["APP_LABEL"],
			})
			if err != nil {
				chaosJob.Status = FailedStatus
				chaosJob.FailedReason = err.Error()
				statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
				return
			}
			logrus.Infof("watching for job %s, run id %v", chaosJob.Name, jobStatusId)

			for event := range w.ResultChan() {
				// check timeout
				if elapsed < int(time.Now().Unix()) {
					logrus.Infof("watch for job %s ended, run id %v", chaosJob.Name, jobStatusId)
					w.Stop()
					return
				}

				if event.Object != nil {
					if reflect.ValueOf(event.Object).Type().Elem().Name() == "Pod" {
						podObject := event.Object.(*coreV1.Pod)
						switch event.Type {
						case watch.Modified:
							if podObject.Status.Phase == coreV1.PodRunning {
								for _, condition := range podObject.Status.Conditions {
									if condition.Type == coreV1.ContainersReady && condition.Status == coreV1.ConditionTrue {
										if podObject.ObjectMeta.DeletionTimestamp == nil {
											// detect newly ready pod
											// if the pod is in the list, just start a new chaos pod
											logrus.Infof("new running pod %s detected, job %s, run id %v",
												podObject.Name, chaosJob.Name, jobStatusId)
											for _, p := range pods {
												if p.Name == podObject.Name {
													if elapsed-int(time.Now().Unix()) > 0 {
														logrus.Infof("scheduling new chaos job for pod %s, job %s, run id %v",
															podObject.Name, chaosJob.Name, jobStatusId)
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
															w.Stop()
															return
														}
														break
													}
												}
											}
											existed := false
											for _, p := range allPods {
												if p.Name == podObject.Name {
													existed = true
													break
												}
											}
											// if not in all pods
											if !existed {
												allPods = append(allPods, *podObject)
												expected := 0
												if percentage == 0 {
													expected = 1
												} else {
													expected = len(allPods) * percentage / 100
												}
												if expected > len(pods) && elapsed-int(time.Now().Unix()) > 0 {
													logrus.Infof("need to add a new job for the increment, scheduling new chaos job for pod %s, job %s, run id %v",
														podObject.Name, chaosJob.Name, jobStatusId)
													// need to scale up
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
														w.Stop()
														return
													}
													pods = append(pods, *podObject)
												}
											}
										}
									}
								}
							}
						case watch.Deleted:
							// detect the deleted pod
							for i := range pods {
								if pods[i].Name == podObject.Name {
									logrus.Infof("new added pod %s detected, job %s, run id %v",
										podObject.Name, chaosJob.Name, jobStatusId)
									// remove from the pods list
									pods = append(pods[:i], pods[i+1:]...)
									break
								}
							}
							for i := range allPods {
								if allPods[i].Name == podObject.Name {
									logrus.Infof("remove %s from the allPods, job %s, run id %v",
										podObject.Name, chaosJob.Name, jobStatusId)
									// remove from the pods list
									allPods = append(allPods[:i], allPods[i+1:]...)
									break
								}
							}
						}
					}
				}
			}
		}()
		// todo maybe this is not very schoen
		time.Sleep(time.Duration(duration) * time.Second)
		// need cleanup here
		logrus.Infof("job %s finished, run id %v, starting cleanup", chaosJob.Name, jobStatusId)
		err := chaosJob.cleanJob(jobStatusId)
		if err != nil {
			logrus.Errorf("job %s cleanup failed, reason: %s", chaosJob.Name, err.Error())
			chaosJob.Status = FailedStatus
			chaosJob.FailedReason = err.Error()
			statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
			return
		}
		logrus.Infof("job %s cleanup done, run id %v", chaosJob.Name, jobStatusId)
		// set status to success
		chaosJob.Status = SuccessStatus
		statusChan <- map[uint]ChaosJob{jobStatusId: *chaosJob}
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
