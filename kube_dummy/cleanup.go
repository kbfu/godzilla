package kube

import (
	"ci-engine/core"
	"fmt"
	types "gitlab.mycyclone.com/testdev/ci-types"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"time"
)

var RemoveChan = make(chan string, 1000)

func CleanUp() {
	for {
		podList, _ := cs.CoreV1().Pods(core.JobNamespace).List(metaV1.ListOptions{})
		for _, p := range podList.Items {
			ci, ok := p.Labels["ci"]
			if ok && ci == "true" {
				if types.Status(p.Status.Phase) == types.Succeeded || types.Status(p.Status.Phase) == types.Failed {
					time.Sleep(time.Minute)
					cs.CoreV1().Pods(core.JobNamespace).Delete(p.Name, &metaV1.DeleteOptions{})
				}
			}
		}
	}
}

func RemoveOne(pipelineId, buildNumber int64) {
	var gracePeriod int64 = 0
	cs.CoreV1().Pods(core.JobNamespace).Delete(fmt.Sprintf("%v-%v", pipelineId, buildNumber),
		&metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
}
