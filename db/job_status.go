package db

type JobStatus struct {
	Base
	ScenarioId uint
	Status     string
}

func (*JobStatus) TableName() string {
	return "job_status"
}

func (j *JobStatus) Add() error {
	return Db.Create(&j).Error
}
