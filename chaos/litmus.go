package chaos

import (
	"fmt"
	"godzilla/env"
	"godzilla/utils"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"strconv"
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
