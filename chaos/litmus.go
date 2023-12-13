/*
 *
 *  * Licensed to the Apache Software Foundation (ASF) under one
 *  * or more contributor license agreements.  See the NOTICE file
 *  * distributed with this work for additional information
 *  * regarding copyright ownership.  The ASF licenses this file
 *  * to you under the Apache License, Version 2.0 (the
 *  * "License"); you may not use this file except in compliance
 *  * with the License.  You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 *
 */

package chaos

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"godzilla/env"
	"godzilla/utils"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"reflect"
	"strconv"
	"strings"
	"time"
)

func (chaosJob *ChaosJob) LitmusJob(jobStatusId uint) batchV1.Job {
	var (
		backOffLimit int32 = 0
		envs         []coreV1.EnvVar
		privileged   = false
	)

	jobName := fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))

	// setup env vars
	for k, v := range chaosJob.Config {
		envs = append(envs, coreV1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	job := batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName,
			Namespace: env.JobNamespace,
			Labels: map[string]string{
				"chaos.job":      "true",
				"chaos.job.id":   fmt.Sprintf("%v", jobStatusId),
				"chaos.job.name": chaosJob.Name,
			},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"chaos.job":      "true",
						"chaos.job.id":   fmt.Sprintf("%v", jobStatusId),
						"chaos.job.name": chaosJob.Name,
					},
				},
				Spec: coreV1.PodSpec{
					ServiceAccountName: chaosJob.ServiceAccountName,
					RestartPolicy:      coreV1.RestartPolicyNever,
					Containers: []coreV1.Container{
						{
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", fmt.Sprintf("./experiments Name %s", chaosJob.Type)},
							Name:            jobName,
							Image:           chaosJob.Image,
							Env:             envs,
							Resources:       coreV1.ResourceRequirements{},
							ImagePullPolicy: coreV1.PullAlways,
							SecurityContext: &coreV1.SecurityContext{
								Privileged: &privileged,
							},
						},
					},
				},
			},
		},
	}
	return job
}

func (chaosJob *ChaosJob) LitmusJobStress(jobStatusId uint, nodeName, podName string) batchV1.Job {
	var (
		backOffLimit int32 = 0
		envs         []coreV1.EnvVar
		privileged         = true
		user         int64 = 0
	)

	termination, _ := strconv.ParseInt(chaosJob.Config["TERMINATION_GRACE_PERIOD_SECONDS"], 10, 64)
	jobName := fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))

	// setup env vars
	for k, v := range chaosJob.Config {
		envs = append(envs, coreV1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	job := batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName,
			Namespace: env.JobNamespace,
			Labels: map[string]string{
				"chaos.job":      "true",
				"chaos.job.id":   fmt.Sprintf("%v", jobStatusId),
				"chaos.job.name": chaosJob.Name,
				"chaos.job.pod":  podName,
			},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"chaos.job":      "true",
						"chaos.job.id":   fmt.Sprintf("%v", jobStatusId),
						"chaos.job.name": chaosJob.Name,
						"chaos.job.pod":  podName,
					},
				},
				Spec: coreV1.PodSpec{
					HostPID:                       true,
					TerminationGracePeriodSeconds: &termination,
					NodeName:                      nodeName,
					Volumes: []coreV1.Volume{
						{
							Name: "socket-path",
							VolumeSource: coreV1.VolumeSource{
								HostPath: &coreV1.HostPathVolumeSource{
									Path: chaosJob.Config["SOCKET_PATH"],
								},
							},
						},
						{
							Name: "sys-path",
							VolumeSource: coreV1.VolumeSource{
								HostPath: &coreV1.HostPathVolumeSource{
									Path: "/sys",
								},
							},
						},
					},
					ServiceAccountName: chaosJob.ServiceAccountName,
					RestartPolicy:      coreV1.RestartPolicyNever,
					Containers: []coreV1.Container{
						{
							VolumeMounts: []coreV1.VolumeMount{
								{
									Name:      "socket-path",
									MountPath: chaosJob.Config["SOCKET_PATH"],
								},
								{
									Name:      "sys-path",
									MountPath: "/sys",
								},
							},
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", "./helpers -name stress-chaos"},
							Name:            jobName,
							Image:           chaosJob.Image,
							Env:             envs,
							Resources:       coreV1.ResourceRequirements{},
							ImagePullPolicy: coreV1.PullAlways,
							SecurityContext: &coreV1.SecurityContext{
								Privileged: &privileged,
								RunAsUser:  &user,
								Capabilities: &coreV1.Capabilities{
									Add: []coreV1.Capability{
										"SYS_ADMIN",
									},
								},
							},
						},
					},
				},
			},
		},
	}
	return job
}

func runLitmusCommon(chaosJob *ChaosJob, jobStatusId uint) {
	job := chaosJob.LitmusJob(jobStatusId)
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
}

func runLitmusStress(chaosJob *ChaosJob, jobStatusId uint) {
	var job batchV1.Job
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
