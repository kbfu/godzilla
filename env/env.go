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

package env

import (
	"github.com/sirupsen/logrus"
	"os"
	"strconv"
)

var (
	JobNamespace          = populateEnv("JOB_NAMESPACE", "test-chaos").(string)
	LocalDebug            = populateEnv("LOCAL_DEBUG", false).(bool)
	LogHouse              = populateEnv("LOG_HOUSE", "github-k8s-runner").(string)
	GithubWorkerName      = populateEnv("ACTIONS_RUNNER_POD_NAME", "").(string)
	GithubWorkDir         = populateEnv("GITHUB_WORKSPACE", "").(string)
	GithubWorkerNamespace = populateEnv("ACTIONS_RUNNER_NAMESPACE", "cicd").(string)
	MysqlUser             = populateEnv("GODZILLA_MYSQL_USER", "root").(string)
	MysqlPassword         = populateEnv("GODZILLA_MYSQL_PASSWORD", "root").(string)
	MysqlHost             = populateEnv("GODZILLA_MYSQL_HOST", "127.0.0.1").(string)
	MysqlPort             = populateEnv("GODZILLA_MYSQL_PORT", "3306").(string)
	MysqlDatabase         = populateEnv("GODZILLA_MYSQL_DATABASE", "godzilla").(string)
)

func populateEnv(name string, defaultValue any) any {
	if name == "LOCAL_DEBUG" {
		if os.Getenv(name) != "" {
			val, err := strconv.ParseBool(os.Getenv("LOCAL_DEBUG"))
			if err != nil {
				logrus.Fatalf("parse LOCAL_DEBUG error, reason, %s", err.Error())
			}
			return val
		}
	} else {
		if os.Getenv(name) != "" {
			return os.Getenv(name)
		}
	}

	return defaultValue
}

func ParseVars() {
	logrus.Info("vars for current run")
	logrus.Infof("LOCAL_DEBUG: %v", LocalDebug)
	logrus.Infof("LOG_HOUSE %s", LogHouse)
	logrus.Infof("ACTIONS_RUNNER_POD_NAME %s", GithubWorkerName)
	logrus.Infof("GITHUB_WORKSPACE %s", GithubWorkDir)
	logrus.Infof("ACTIONS_RUNNER_NAMESPACE %s", GithubWorkerNamespace)
}
