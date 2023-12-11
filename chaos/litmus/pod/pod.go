package pod

import (
	_ "embed"
	"github.com/sirupsen/logrus"
	"k8s.io/apimachinery/pkg/util/yaml"
)

type PodDelete struct {
	Image              string            `yaml:"image"`
	ServiceAccountName string            `yaml:"serviceAccountName"`
	Env                map[string]string `yaml:"env"`
}

var (
	//go:embed common.yaml
	common []byte
	//go:embed pod-delete.yaml
	deletePod []byte
)

func PopulateDefaultDeletePod() (config PodDelete) {
	var (
		commonConfig PodDelete
		deleteConfig PodDelete
	)
	err := yaml.Unmarshal(common, &commonConfig)
	if err != nil {
		logrus.Fatalf("unmarshal default pod common yaml file failed, reason: %s", err.Error())
	}
	err = yaml.Unmarshal(deletePod, &deleteConfig)
	if err != nil {
		logrus.Fatalf("unmarshal pod delete yaml file failed, reason: %s", err.Error())
	}
	config = commonConfig
	if config.Env == nil {
		config.Env = make(map[string]string)
	}
	if deleteConfig.Image != "" {
		config.Image = deleteConfig.Image
	}
	for k, v := range deleteConfig.Env {
		config.Env[k] = v
	}
	if deleteConfig.ServiceAccountName != "" {
		config.ServiceAccountName = deleteConfig.ServiceAccountName
	}

	return config
}
