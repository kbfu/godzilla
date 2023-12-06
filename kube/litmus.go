package kube

import (
	"fmt"
	"godzilla/utils"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
)

type litmusType string

const (
	podDelete litmusType = "pod-delete"
)

func (chaosJob *ChaosJob) chaosEnvs(targetNamespace string) []coreV1.EnvVar {
	var envs []coreV1.EnvVar
	switch chaosJob.Type {
	case string(podDelete):
		envs = []coreV1.EnvVar{
			{
				Name:  "APP_LABEL",
				Value: chaosJob.Label,
			},
			{
				Name:  "LIB",
				Value: "litmus",
			},
			{
				Name: "TARGET_PODS",
			},
			{
				Name: "NODE_LABEL",
			},
			{
				Name:  "STATUS_CHECK_TIMEOUT",
				Value: "180",
			},
			{
				Name:  "TOTAL_CHAOS_DURATION",
				Value: chaosJob.Duration,
			},
			{
				Name:  "FORCE",
				Value: "false",
			},
			{
				Name:  "EXPERIMENT_NAME",
				Value: "pod-delete",
			},
			{
				Name:  "ANNOTATION_KEY",
				Value: "litmuschaos.io/chaos",
			},
			{
				Name:  "ANNOTATION_CHECK",
				Value: "false",
			},
			{
				Name:  "APP_NAMESPACE",
				Value: targetNamespace,
			},
			{
				Name:  "CHAOS_SERVICE_ACCOUNT",
				Value: "chaos-admin",
			},
			{
				Name:  "CHAOS_INTERVAL",
				Value: chaosJob.Interval,
			},
			{
				Name:  "SEQUENCE",
				Value: "parallel",
			},
			{
				Name: "JOB_CLEANUP_POLICY",
			},
			{
				Name:  "STATUS_CHECK_DELAY",
				Value: "2",
			},
			Name: CHAOS_NAMESPACE
			Value: litmus
			Name: LIB_IMAGE_PULL_POLICY
			Value: Always
			Name: TERMINATION_GRACE_PERIOD_SECONDS
			Value: '0'
			Name: RAMP_TIME
			Name: PODS_AFFECTED_PERC
			Name: POD_NAME
			ValueFrom: fieldRef, :
			apiVersion: v1
			fieldPath: metadata.name
		},
	}
}
return envs
}

func (chaosJob *ChaosJob) LitmusJob() (job batchV1.Job, err error) {
	namespace := ""
	jobName := ""
	if chaosJob.Namespace == "" {
		data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return
		}
		namespace = string(data)
	} else {
		namespace = chaosJob.Namespace
	}
	jobName = fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))
	var backOffLimit int32 = 0

	job = batchV1.Job{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      jobName,
			Namespace: namespace,
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
					Containers: []coreV1.Container{
						{
							Command:         []string{"/bin/bash"},
							Args:            []string{"-c", fmt.Sprintf("./experiments Name %s", chaosJob.Type)},
							Name:            "",
							Image:           "wxt432/chaos-go-runner:latest",
							Env:             envs,
							Resources:       coreV1.ResourceRequirements{},
							ImagePullPolicy: coreV1.PullAlways,
							SecurityContext: &coreV1.SecurityContext{
								Capabilities:             nil,
								Privileged:               nil,
								SELinuxOptions:           nil,
								WindowsOptions:           nil,
								RunAsUser:                nil,
								RunAsGroup:               nil,
								RunAsNonRoot:             nil,
								ReadOnlyRootFilesystem:   nil,
								AllowPrivilegeEscalation: nil,
								ProcMount:                nil,
								SeccompProfile:           nil,
							},
						},
					},
				},
			},
		},
	}
	return
}
