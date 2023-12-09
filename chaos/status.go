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

func initStatus(chaosJobs [][]ChaosJob, scenarioId uint) error {
	jobs, _ := yaml.Marshal(chaosJobs)
	jobStatus := db.JobStatus{
		ScenarioId: scenarioId,
		Status:     string(jobs),
	}
	return jobStatus.Add()
}
