package chaos

import (
	"godzilla/db"
	"gopkg.in/yaml.v2"
)

type JobStatus string

const (
	PendingStatus JobStatus = "pending"
	RunningStatus JobStatus = "running"
	SuccessStatus JobStatus = "success"
	FailedStatus  JobStatus = "failed"
)

var statusChan = make(chan ChaosJob, 100)

func initStatus(chaosJobs [][]ChaosJob, scenarioId uint) (statusId uint, err error) {
	jobs, _ := yaml.Marshal(chaosJobs)
	jobStatus := db.JobStatus{
		ScenarioId: scenarioId,
		Status:     string(jobs),
	}
	err = jobStatus.Add()
	return jobStatus.Id, err
}
