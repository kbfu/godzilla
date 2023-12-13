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
	"github.com/sirupsen/logrus"
	"godzilla/env"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

var (
	client *kubernetes.Clientset
	config *rest.Config
)

func InitKubeClient() {
	if client == nil {
		fetchConfig()
	}
}

func ReadyChaosEnv(namespace string) {
	// setup rbac
	err := addClusterRole()
	if err != nil {
		logrus.Fatalf("create cluster role failed, reason: %s", err.Error())
	}
	err = addServiceAccount(namespace)
	if err != nil {
		logrus.Fatalf("create service account failed, reason: %s", err.Error())
	}
	err = addRoleBinding(namespace)
	if err != nil {
		logrus.Fatalf("create role binding failed, reason: %s", err.Error())
	}
}

func fetchConfig() {
	var err error
	if env.LocalDebug {
		config, err = clientcmd.BuildConfigFromFlags("", homedir.HomeDir()+"/.kube/config")
		if err != nil {
			logrus.Fatalf("get config error, reason: %s", err.Error())
		}
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Fatalf("get client set error, reason: %s", err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("get in cluster config error, reason: %s", err.Error())
		}
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Fatalf("get in cluster client set error, reason: %s", err.Error())
		}
	}

}
