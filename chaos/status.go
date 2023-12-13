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
	"godzilla/db"
	"gopkg.in/yaml.v2"
	"time"
)

type JobStatus string

// pending -> running -> success -> failed
//
//				     \-> failed
//	                 \-> unknown
const (
	PendingStatus JobStatus = "pending"
	RunningStatus JobStatus = "running"
	SuccessStatus JobStatus = "success"
	FailedStatus  JobStatus = "failed"
	UnknownStatus JobStatus = "unknown"
)

var statusChan = make(chan map[uint]ChaosJob, 100)

func statusCheck(prev JobStatus, curr JobStatus) bool {
	if prev == PendingStatus && (curr == RunningStatus || curr == FailedStatus || curr == UnknownStatus || curr == SuccessStatus) {
		return true
	} else if prev == RunningStatus && (curr == FailedStatus || curr == UnknownStatus || curr == SuccessStatus) {
		return true
	} else if prev == SuccessStatus && curr == FailedStatus {
		return true
	}
	return false
}

func StatusWorker() {
	for status := range statusChan {
		for k, v := range status {
			jobStatus := db.JobStatus{Base: db.Base{Id: k}}
			err := jobStatus.GetById()
			if err != nil {
				logrus.Errorf("update status failed for id %v, reason: %s", k, err.Error())
				break
			}
			var chaosJobs [][]ChaosJob
			err = yaml.Unmarshal([]byte(jobStatus.Status), &chaosJobs)
			if err != nil {
				logrus.Errorf("update status failed for id %v, reason: %s", k, err.Error())
				break
			}
		out:
			for i := range chaosJobs {
				for j := range chaosJobs[i] {
					if chaosJobs[i][j].Name == v.Name {
						if statusCheck(chaosJobs[i][j].Status, v.Status) {
							chaosJobs[i][j].Status = v.Status
							chaosJobs[i][j].FailedReason = v.FailedReason
							break out
						}
					}
				}
			}
			// update status
			data, _ := yaml.Marshal(chaosJobs)
			jobStatus.UpdatedAt = time.Now()
			jobStatus.Status = string(data)
			err = jobStatus.UpdateById()
			if err != nil {
				logrus.Errorf("update status failed for id %v, reason: %s", k, err.Error())
				break
			}
			logrus.Infof("status updated for id %v", k)
		}
	}
}

func initStatus(chaosJobs [][]ChaosJob, scenarioId uint) (statusId uint, err error) {
	jobs, _ := yaml.Marshal(chaosJobs)
	jobStatus := db.JobStatus{
		ScenarioId: scenarioId,
		Status:     string(jobs),
	}
	err = jobStatus.Add()
	return jobStatus.Id, err
}

func initStatusOne(chaosJobs [][]ChaosJob) (statusId uint, err error) {
	jobs, _ := yaml.Marshal(chaosJobs)
	jobStatus := db.JobStatus{
		Status: string(jobs),
	}
	err = jobStatus.Add()
	return jobStatus.Id, err
}
