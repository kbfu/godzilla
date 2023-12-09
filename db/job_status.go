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

func (j *JobStatus) GetById() error {
	return Db.Where("id = ?", j.Id).Find(&j).Error
}

func (j *JobStatus) UpdateById() error {
	return Db.Model(&j).Updates(JobStatus{
		Base: Base{
			UpdatedAt: j.UpdatedAt,
		},
		Status: j.Status,
	}).Error
}
