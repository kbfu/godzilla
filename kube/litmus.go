package kube

import (
	"fmt"
	"godzilla/chaos/litmus/pod"
	"godzilla/types"
	"godzilla/utils"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

func (chaosJob *ChaosJob) LitmusJob() batchV1.Job {
	var (
		backOffLimit   int32 = 0
		envs           []coreV1.EnvVar
		privileged     = false
		image          = ""
		serviceAccount = ""
	)

	jobName := fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))

	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		config := pod.PopulateDefaultDeletePod()
		// override default config
		for k, v := range chaosJob.Config {
			config.Env[k] = v
		}
		// allow it to be overridden by env vars
		for k := range config.Env {
			if os.Getenv(k) != "" {
				config.Env[k] = os.Getenv(k)
			}
		}

		// setup env vars
		for k, v := range config.Env {
			envs = append(envs, coreV1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
		image = config.Image
		if chaosJob.Image != "" {
			image = chaosJob.Image
		}
		serviceAccount = config.ServiceAccountName
		if chaosJob.ServiceAccountName != "" {
			serviceAccount = chaosJob.ServiceAccountName
		}
	}

	job := batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName,
			Namespace: chaosJob.Namespace,
			Labels: map[string]string{
				"chaos.job": "true",
			},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"chaos.job": "true",
					},
				},
				Spec: coreV1.PodSpec{
					ServiceAccountName: serviceAccount,
					RestartPolicy:      coreV1.RestartPolicyNever,
					Containers: []coreV1.Container{
						{
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", fmt.Sprintf("./experiments Name %s", chaosJob.Type)},
							Name:            jobName,
							Image:           image,
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
