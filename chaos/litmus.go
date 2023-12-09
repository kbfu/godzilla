package chaos

import (
	"fmt"
	"godzilla/types"
	"godzilla/utils"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (chaosJob *ChaosJob) LitmusJob(jobStatusId uint) batchV1.Job {
	var (
		backOffLimit int32 = 0
		envs         []coreV1.EnvVar
		privileged   = false
	)

	jobName := fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))

	switch chaosJob.Type {
	case string(types.LitmusPodDelete):
		// setup env vars
		for k, v := range chaosJob.Config {
			envs = append(envs, coreV1.EnvVar{
				Name:  k,
				Value: v,
			})
		}
	}

	job := batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName,
			Namespace: chaosJob.Namespace,
			Labels: map[string]string{
				"chaos.job":    "true",
				"chaos.job.id": fmt.Sprintf("%v", jobStatusId),
			},
		},
		Spec: batchV1.JobSpec{
			BackoffLimit: &backOffLimit,
			Template: coreV1.PodTemplateSpec{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						"chaos.job":    "true",
						"chaos.job.id": fmt.Sprintf("%v", jobStatusId),
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
