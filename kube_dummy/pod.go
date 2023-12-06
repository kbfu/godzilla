package kube

import (
	"ci-engine/console"
	"ci-engine/core"
	"fmt"
	"gitlab.mycyclone.com/testdev/ci-types"
	"k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"reflect"
	"time"

	//"reflect"
	"runtime/debug"
)

func CreatePod(pod *v1.Pod) error {
	_, err := cs.CoreV1().Pods(core.JobNamespace).Create(pod)
	return err
}

// FetchStatus 实时更新任务状态使用
// Fail的状态会立刻删除整个pod的运行
func FetchStatus(job types.Job, jobStatus *types.JobStatus) {
	w, _ := cs.CoreV1().Pods(core.JobNamespace).Watch(metaV1.ListOptions{
		LabelSelector: "ci=true",
		FieldSelector: fmt.Sprintf("metadata.name=%v-%v", job.PipelineId, job.BuildNumber),
		Watch:         true,
	})
	for c := range w.ResultChan() {
		if reflect.ValueOf(c.Object).Type().Elem().Name() == "Pod" {
			pod := c.Object.(*v1.Pod)
			for _, cs := range pod.Status.ContainerStatuses {
				if cs.Image != "nexus-docker.mycyclone.com/testdev/dummy:latest" {
					status := ""
					if cs.State.Waiting != nil {
						status = string(types.Pending)
					} else if cs.State.Running != nil {
						status = string(types.Running)
					} else if cs.State.Terminated != nil {
						switch cs.State.Terminated.ExitCode {
						case 0:
							status = string(types.Succeeded)
						default:
							status = string(types.Failed)
							RemoveOne(job.PipelineId, job.BuildNumber)
						}
					}

					for i := range jobStatus.Steps {
						if string(jobStatus.Steps[i].Status) != status && jobStatus.Steps[i].Name == cs.Name {
							jobStatus.Steps[i].Status = types.Status(status)
							console.SendStepStatus(types.StatusNotification{
								PipelineId:  job.PipelineId,
								BuildNumber: job.BuildNumber,
								StepName:    cs.Name,
								StepStatus:  types.Status(status),
								OpenIds:     nil,
								Branch:      job.Envs["OUROBOROS_GIT_REF"],
								Commit:      job.Envs["OUROBOROS_GIT_COMMIT"],
								Url:         "",
							})
							if jobStatus.Steps[i].Status == types.Failed {
								jobStatus.Status = types.Failed
								console.SendStatus(jobStatus)
								w.Stop()
								break
							} else if jobStatus.Steps[i].Status == types.Canceled || jobStatus.Steps[i].Status == types.Skipped {
								jobStatus.Status = types.Canceled
								console.SendStatus(jobStatus)
								w.Stop()
								break
							} else if i == len(jobStatus.Steps)-1 && jobStatus.Steps[i].Status == types.Succeeded {
								jobStatus.Status = types.Succeeded
								console.SendStatus(jobStatus)
								w.Stop()
								break
							} else {
								jobStatus.Status = types.Running
								console.SendStatus(jobStatus)
								break
							}
						}
					}
				}
			}
		}
	}
}

// Run 将任务并生成pod spec使用kube api进行调度运行
// 除了首步会立刻执行，其他的容器都是使用dummy image来阻塞，需要运行的时候再更新真正的image后由k8s重启调度
// 所有容器的workdir都固定在/src路径下，所有容器会共享这个路径下的文件系统
// Job类型中有一个IgnoreError的field，目前都会保持默认值false，也就是说只要一步出错会导致任务完全停止
func Run(j types.Job) {
	var containers []v1.Container
	resources := v1.ResourceRequirements{}
	if core.JobMemoryLimits != "" {
		resources = v1.ResourceRequirements{
			Limits: v1.ResourceList{
				"memory": resource.MustParse(core.JobMemoryLimits),
			},
			Requests: v1.ResourceList{
				"memory": resource.MustParse(core.JobMemoryRequests),
			},
		}
	}

	var envs []v1.EnvVar
	for k, v := range j.Envs {
		envs = append(envs, v1.EnvVar{
			Name:  k,
			Value: v,
		})
	}

	volumes := []v1.Volume{
		{
			Name: "src",
			VolumeSource: v1.VolumeSource{
				EmptyDir: &v1.EmptyDirVolumeSource{},
			},
		},
	}

	for i, s := range j.Steps {
		var securityContext v1.SecurityContext
		switch s.Artifact {
		case types.DockerArtifact:
			securityContext = v1.SecurityContext{
				Privileged: func() *bool { t := true; return &t }(),
			}
		}

		volumeMounts := []v1.VolumeMount{
			{
				MountPath: "/src",
				Name:      "src",
			},
		}
		// set additional configs for container, for sth you don't wanna be seen from gits
		imageConfigs, err := console.GetConfig(types.ImageConfigs)
		if err != nil {
			logger.Error(err)
			logger.Error(string(debug.Stack()))
			return
		}
		for _, config := range imageConfigs.(types.ImageConfigsType) {
			if config.Image == s.Image {
				for _, env := range config.Envs {
					envs = append(envs, v1.EnvVar{
						Name:  env.Key,
						Value: env.Value,
					})
				}
				for _, volume := range config.Volumes {
					switch volume.Type {
					case types.Secret:
						var mode int32 = 0400
						optional := false
						volumes = append(volumes, v1.Volume{
							Name: volume.Name,
							VolumeSource: v1.VolumeSource{
								Secret: &v1.SecretVolumeSource{
									SecretName:  volume.Name,
									DefaultMode: &mode,
									Optional:    &optional,
								},
							},
						})
						volumeMounts = append(volumeMounts, v1.VolumeMount{
							MountPath: volume.MountPath,
							Name:      volume.Name,
						})
					}
				}
			}
		}

		// accept empty command
		var (
			command []string
			args    []string
		)
		if s.Command != "" {
			command = []string{"/bin/sh", "-c", "-x", "-e"}
			args = []string{fmt.Sprintf("%s", s.Command)}
		}
		if i == 0 {
			containers = append(containers, v1.Container{
				Image: s.Image,
				Env: append(envs, v1.EnvVar{
					Name:  "OUROBOROS_STEP_NAME",
					Value: s.Name,
				}),
				Command:         command,
				TTY:             true,
				Stdin:           true,
				Args:            args,
				ImagePullPolicy: "Always",
				Name:            s.Name,
				Resources:       resources,
				WorkingDir:      "/src",
				VolumeMounts:    volumeMounts,
				SecurityContext: &securityContext,
			})
		} else {
			containers = append(containers, v1.Container{
				Image: "nexus-docker.mycyclone.com/testdev/dummy:latest",
				Env: append(envs, v1.EnvVar{
					Name:  "OUROBOROS_STEP_NAME",
					Value: s.Name,
				}),
				Command:         command,
				Args:            args,
				TTY:             true,
				Stdin:           true,
				ImagePullPolicy: "Always",
				Name:            s.Name,
				Resources:       resources,
				WorkingDir:      "/src",
				VolumeMounts:    volumeMounts,
				SecurityContext: &securityContext,
			})
		}
	}

	var seconds int64 = 300
	pod := v1.Pod{
		ObjectMeta: metaV1.ObjectMeta{
			Name:      fmt.Sprintf("%v-%v", j.PipelineId, j.BuildNumber),
			Namespace: core.JobNamespace,
			Labels: map[string]string{
				"ci": "true",
			},
		},
		Spec: v1.PodSpec{
			Containers: containers,
			DNSPolicy:  "ClusterFirst",
			ImagePullSecrets: []v1.LocalObjectReference{
				{
					Name: core.PullSecrets,
				},
			},
			Tolerations: []v1.Toleration{
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/not-ready",
					Operator:          "Exists",
					TolerationSeconds: &seconds,
				},
				{
					Effect:            "NoExecute",
					Key:               "node.kubernetes.io/unreachable",
					Operator:          "Exists",
					TolerationSeconds: &seconds,
				},
			},
			RestartPolicy: "Never",
			Volumes:       volumes,
		},
	}

	err := CreatePod(&pod)
	if err != nil {
		logger.Error(err)
		logger.Error(string(debug.Stack()))
		return
	}

	jobStatus := &types.JobStatus{
		PipelineId:  j.PipelineId,
		BuildNumber: j.BuildNumber,
		Status:      types.Pending,
		Steps:       []types.StepStatus{},
		Envs:        j.Envs,
	}
	var stepStatus []types.StepStatus
	for _, step := range j.Steps {
		stepStatus = append(stepStatus, types.StepStatus{
			Status: types.Pending,
			Name:   step.Name,
		})
	}
	jobStatus.Steps = stepStatus

	go FetchStatus(j, jobStatus)

	for i, s := range j.Steps {
	outer:
		for {
			select {
			case podName := <-RemoveChan:
				RemoveChan <- podName
				if podName == fmt.Sprintf("%v-%v", j.PipelineId, j.BuildNumber) {
					break outer
				}
			default:
			}
			pods, err := cs.CoreV1().Pods(core.JobNamespace).List(metaV1.ListOptions{
				LabelSelector: "ci=true",
				FieldSelector: fmt.Sprintf("metadata.name=%v-%v,", j.PipelineId, j.BuildNumber),
			})
			if err != nil {
				logger.Warning(string(debug.Stack()))
				logger.Warning(err)
			}
			if len(pods.Items) > 0 {
				if len(pods.Items[0].Spec.Containers) > 0 && pods.Items[0].Status.Phase == v1.PodRunning {
					if i != 0 {
						switch jobStatus.Steps[i-1].Status {
						case types.Failed, types.Canceled, types.Skipped:
							if !s.IgnoreError {
								jobStatus.Steps[i].Status = types.Skipped
								break outer
							} else {
								pods.Items[0].Spec.Containers[i].Image = s.Image
								_, err = cs.CoreV1().Pods(core.JobNamespace).Update(&pods.Items[0])
								if err != nil {
									logger.Warning(string(debug.Stack()))
									logger.Warning(err)
								} else {
									break outer
								}
							}
						case types.Running:
						case types.Succeeded:
							pods.Items[0].Spec.Containers[i].Image = s.Image
							_, err = cs.CoreV1().Pods(core.JobNamespace).Update(&pods.Items[0])
							if err != nil {
								logger.Warning(string(debug.Stack()))
								logger.Warning(err)
							} else {
								break outer
							}
						}
					} else {
						break outer
					}
				}
				time.Sleep(time.Second)
			}
		}
		RetrieveLogs(j, s.Name, jobStatus)
	}

	pods, err := cs.CoreV1().Pods(core.JobNamespace).List(metaV1.ListOptions{
		LabelSelector: "ci=true",
		FieldSelector: fmt.Sprintf("metadata.name=%v-%v,", j.PipelineId, j.BuildNumber),
	})
	for _, p := range pods.Items {
		time.Sleep(time.Minute)
		cs.CoreV1().Pods(core.JobNamespace).Delete(p.Name, &metaV1.DeleteOptions{})
	}
}
