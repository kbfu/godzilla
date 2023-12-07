package kube

import (
	"github.com/sirupsen/logrus"
	"godzilla/core"
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
	if core.LocalDebug {
		config, err = clientcmd.BuildConfigFromFlags("", homedir.HomeDir()+"/.kube/config")
		if err != nil {
			logrus.Fatalf("Get config error, reason: %s", err.Error())
		}
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Fatalf("Get client set error, reason: %s", err.Error())
		}
	} else {
		config, err = rest.InClusterConfig()
		if err != nil {
			logrus.Fatalf("Get in cluster config error, reason: %s", err.Error())
		}
		client, err = kubernetes.NewForConfig(config)
		if err != nil {
			logrus.Fatalf("Get in cluster client set error, reason: %s", err.Error())
		}
	}

}
