package kube

import (
	"ci-engine/core"
	"ci-engine/minio"
	"fmt"
	"gitlab.mycyclone.com/testdev/ci-types"
	"io"
	v1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"reflect"
	"runtime/debug"
	"time"
)

// RetrieveLogs 根据job从k8s的相应pod的容器中获得stdout中打印的字符串
func RetrieveLogs(j types.Job, containerName string, jobStatus *types.JobStatus) {
	pod := cs.CoreV1().Pods(core.JobNamespace)
	w, _ := pod.Watch(metaV1.ListOptions{
		LabelSelector: "ci=true",
		FieldSelector: fmt.Sprintf("metadata.name=%v-%v,", j.PipelineId, j.BuildNumber),
	})

	for c := range w.ResultChan() {
		if c.Object != nil && c.Type != watch.Error && c.Type != watch.Deleted {
			if reflect.ValueOf(c.Object).Type().Elem().Name() == "Pod" {
				podObject := c.Object.(*v1.Pod)
				for _, cs := range podObject.Status.ContainerStatuses {
					if cs.Name == containerName {
						select {
						// 检查是否被取消任务，如果是则返回
						case podName := <-RemoveChan:
							if podName == fmt.Sprintf("%v-%v", j.PipelineId, j.BuildNumber) {
								jobStatus.Status = types.Canceled
								for i := range jobStatus.Steps {
									if jobStatus.Steps[i].Name == cs.Name {
										jobStatus.Steps[i].Status = types.Canceled
										break
									}
								}
								return
							} else {
								RemoveChan <- podName
							}
						default:
						}

						// 检查是不是dummy image，如果是的话需要等调度
						if cs.Image != "nexus-docker.mycyclone.com/testdev/dummy:latest" {
							logs := pod.GetLogs(fmt.Sprintf("%v-%v", j.PipelineId, j.BuildNumber),
								&v1.PodLogOptions{Follow: true, Container: containerName, Timestamps: true})

							reader, err := logs.Stream()
							if err != nil {
								logger.Warning(err)
								return
							}

							// 任务还在运行时的状态时，日志会暂存在本地，提高IO效率
							objectName := fmt.Sprintf("logs/%v-%v-%s.log", j.PipelineId, j.BuildNumber, containerName)
							f, err := os.OpenFile(objectName, os.O_CREATE|os.O_RDWR, 0644)
							if err != nil {
								logger.Error(string(debug.Stack()))
								logger.Error(err)
								return
							}
							// 初始化指针
							offset := 0
							// 三秒获取一次
							t := time.NewTicker(3 * time.Second)

							for {
								// 初始化一次读取长度一兆字节
								buf := make([]byte, 1024*1024)
								numBytes, err := reader.Read(buf)
								if err == io.EOF {
									// stdout返回EOF，结束，然后最终上传日志到minio
									t.Stop()
									w.Stop()
									minio.FPutObject(objectName, objectName)
									os.Remove(objectName)
									break
								}
								if err != nil {
									// 出现预料之外的错误，结束，然后最终上传日志到minio
									t.Stop()
									w.Stop()
									minio.FPutObject(objectName, objectName)
									os.Remove(objectName)
									logger.Error(string(debug.Stack()))
									logger.Error(err)
									return
								}
								// 写入文件，如果没读满buf，则只写入读取到的部分
								_, err = f.WriteAt(buf[:numBytes], int64(offset))
								if err != nil {
									logger.Error(string(debug.Stack()))
									logger.Error(err)
									return
								}
								offset += numBytes
								go func() {
									select {
									case <-t.C:
										// 每隔三秒钟上传一次到minio，保证数据不丢失
										minio.FPutObject(objectName, objectName)
									}
								}()
							}
						}
					}
				}
			}
		}
	}
}
