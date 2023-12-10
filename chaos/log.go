package chaos

import (
	"context"
	"fmt"
	"github.com/sirupsen/logrus"
	"godzilla/env"
	"io"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"os"
	"reflect"
	"strings"
	"time"
)

func (chaosJob *ChaosJob) fetchChaosLogs(actualName string) {
	pod := client.CoreV1().Pods(env.JobNamespace)
	w, err := pod.Watch(context.TODO(), metaV1.ListOptions{
		LabelSelector: "chaos.job=true",
	})
	if err != nil {
		logrus.Error(err)
		return
	}

	for c := range w.ResultChan() {
		if c.Object != nil && c.Type != watch.Error && c.Type != watch.Deleted {
			if reflect.ValueOf(c.Object).Type().Elem().Name() == "Pod" {
				podObject := c.Object.(*coreV1.Pod)
				if strings.Contains(podObject.Name, actualName) {
					for _, cs := range podObject.Status.ContainerStatuses {
						if cs.State.Waiting != nil {
							continue
						}
						logs := pod.GetLogs(podObject.ObjectMeta.Name,
							&coreV1.PodLogOptions{Follow: true, Container: cs.Name, Timestamps: true})

						reader, err := logs.Stream(context.TODO())
						if err != nil {
							logrus.Warning(err)
							return
						}

						os.Mkdir("logs", 0755)
						objectName := fmt.Sprintf("logs/%s.log", actualName)
						f, err := os.OpenFile(objectName, os.O_CREATE|os.O_RDWR, 0644)
						if err != nil {
							logrus.Fatal(err)
						}
						offset := 0
						t := time.NewTicker(3 * time.Second)

						for {
							buf := make([]byte, 1024*1024)
							numBytes, err := reader.Read(buf)
							if err == io.EOF {
								t.Stop()
								w.Stop()
								break
							}
							if err != nil {
								t.Stop()
								w.Stop()
								logrus.Error(err)
								return
							}
							_, err = f.WriteAt(buf[:numBytes], int64(offset))
							if err != nil {
								logrus.Error(err)
								return
							}
							offset += numBytes
						}
					}
				}
			}
		}
	}
}
