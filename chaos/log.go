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
	w, err := client.CoreV1().Pods(env.JobNamespace).Watch(context.TODO(), metaV1.ListOptions{
		LabelSelector: "chaos.job=true",
	})
	if err != nil {
		logrus.Error(err)
		return
	}

	for event := range w.ResultChan() {
		if event.Object != nil && event.Type != watch.Error && event.Type != watch.Deleted {
			if reflect.ValueOf(event.Object).Type().Elem().Name() == "Pod" {
				podObject := event.Object.(*coreV1.Pod)
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
							w.Stop()
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
								w.Stop()
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
