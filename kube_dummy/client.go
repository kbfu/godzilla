package kube

import (
	"ci-engine/core"
	"github.com/op/go-logging"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	cs     *kubernetes.Clientset
	logger = logging.MustGetLogger("kube")
	config = &rest.Config{
		Host:        core.KubeApi,
		BearerToken: core.KubeToken,
		TLSClientConfig: rest.TLSClientConfig{
			Insecure: true,
		},
	}
)

func InitClient() error {
	clientSet, err := kubernetes.NewForConfig(config)
	if err != nil {
		return err
	}
	cs = clientSet
	return nil
}
