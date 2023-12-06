package kube

import (
	"godzilla/types"
	"gopkg.in/yaml.v3"
	"io/fs"
	"os"
	"path/filepath"
	"sync"
)

type ChaosJob struct {
	Name string          `yaml:"name"`
	Type types.ChaosType `yaml:"type"`
	podJob
}

type podJob struct {
	Namespace string `yaml:"namespace"`
	Label     string `yaml:"label"`
	Interval  string `yaml:"interval"`
	Duration  string `yaml:"duration"`
}

func (chaosJob *ChaosJob) Run() {

}

func CreateChaos() error {
	var (
		chaosJobs [][]ChaosJob
		wg        sync.WaitGroup
	)
	err := filepath.Walk("scenarios", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			err = yaml.Unmarshal(data, &chaosJobs)
			if err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return err
	}
	for _, parallelJobs := range chaosJobs {
		for _, j := range parallelJobs {
			wg.Add(1)
			j := j
			go func() {
				j.Run()
				// todo need to watch chaos jobs here
				// todo collect logs to files
				// todo copy back to github actions worker if needed
				// todo remove all chaos pods if time elapsed
				wg.Done()
			}()
		}
		wg.Wait()
	}
	return nil
	//client.BatchV1().Jobs(namespace).Create(context.Background(), &batchV1.Job{
	//	TypeMeta:   metaV1.TypeMeta{},
	//	ObjectMeta: metaV1.ObjectMeta{},
	//	//Spec:       v1.JobSpec{},
	//	//Status:     v1.JobStatus{},
	//}, metaV1.CreateOptions{})
}
