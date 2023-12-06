package kube

import (
	"context"
	"fmt"
	"godzilla/utils"
	"gopkg.in/yaml.v3"
	"io/fs"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	"path/filepath"
	"sync"
)

type ChaosJob struct {
	Name string `yaml:"name"`
	Type string `yaml:"type"`
	podJob
}

type podJob struct {
	Namespace string `yaml:"namespace"`
	Label     string `yaml:"label"`
	Interval  string `yaml:"interval"`
	Duration  string `yaml:"duration"`
}

func (chaosJob *ChaosJob) Run() error {
	namespace := ""
	jobName := ""
	if chaosJob.Namespace == "" {
		data, err := os.ReadFile("/var/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			return err
		}
		namespace = string(data)
	} else {
		namespace = chaosJob.Namespace
	}
	jobName = fmt.Sprintf("%s-%s", chaosJob.Name, utils.RandomString(10))
	var backOffLimit int32 = 0

	_, err := client.BatchV1().Jobs(namespace).Create(context.Background(), &batchV1.Job{
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
				ObjectMeta: metaV1.ObjectMeta{},
				Spec: coreV1.PodSpec{
					Containers: []coreV1.Container{
						{
							Name:            "",
							Image:           "wxt432/chaos-go-runner:latest",
							Env:             nil,
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
	}, metaV1.CreateOptions{})
	return err
}

func CreateChaos() error {
	var (
		chaosJobs [][]ChaosJob
		wg        sync.WaitGroup
	)
	err := filepath.Walk("scenarios", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(data, &chaosJobs)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			wg.Add(1)
			j := j
			go func() {
				j.Run()
				// todo need to watch chaos jobs here
				// todo collect logs to files
				// todo copy back to github actions worker if needed
				// todo remove all chaos pods if time elapsed
				wg.Done()
			}()
		}
		wg.Wait()
	}
	return nil
	//client.BatchV1().Jobs(namespace).Create(context.Background(), &batchV1.Job{
	//	TypeMeta:   metaV1.TypeMeta{},
	//	ObjectMeta: metaV1.ObjectMeta{},
	//	//Spec:       v1.JobSpec{},
	//	//Status:     v1.JobStatus{},
	//}, metaV1.CreateOptions{})
}
