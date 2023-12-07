package kube

import (
	"context"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"os"
)

func copyIntoPod(podName string, namespace string, srcPath string, dstPath string) {
	localFile, err := os.Open(srcPath)
	if err != nil {
		logrus.Errorf("Error opening local file: %s", err)
		return
	}
	defer localFile.Close()

	// Create a stream to the container
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec")

	req.VersionedParams(&coreV1.PodExecOptions{
		Command: []string{"sh", "-c", "cat > " + dstPath},
		Stdin:   true,
		Stdout:  true,
		Stderr:  true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		logrus.Errorf("Error creating executor: %s", err)
		return
	}

	// Create a stream to the container
	err = exec.StreamWithContext(context.TODO(), remotecommand.StreamOptions{
		Stdin:  localFile,
		Stdout: os.Stdout,
		Stderr: os.Stderr,
		Tty:    false,
	})
	if err != nil {
		logrus.Errorf("Error streaming: %s", err)
		return
	}

	logrus.Infof("File %s copied successfully", srcPath)
}
