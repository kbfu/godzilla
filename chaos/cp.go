/*
 *
 *  * Licensed to the Apache Software Foundation (ASF) under one
 *  * or more contributor license agreements.  See the NOTICE file
 *  * distributed with this work for additional information
 *  * regarding copyright ownership.  The ASF licenses this file
 *  * to you under the Apache License, Version 2.0 (the
 *  * "License"); you may not use this file except in compliance
 *  * with the License.  You may obtain a copy of the License at
 *  *
 *  *     http://www.apache.org/licenses/LICENSE-2.0
 *  *
 *  * Unless required by applicable law or agreed to in writing, software
 *  * distributed under the License is distributed on an "AS IS" BASIS,
 *  * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 *  * See the License for the specific language governing permissions and
 *  * limitations under the License.
 *
 *
 */

package chaos

import (
	"context"
	"github.com/sirupsen/logrus"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/client-go/kubernetes/scheme"
	"k8s.io/client-go/tools/remotecommand"
	"os"
)

func copyIntoPod(podName string, namespace string, container string, srcPath string, dstPath string) {
	localFile, err := os.Open(srcPath)
	if err != nil {
		logrus.Errorf("error opening local file: %s", err)
		return
	}
	defer localFile.Close()

	// Create a stream to the container
	req := client.CoreV1().RESTClient().Post().
		Resource("pods").
		Name(podName).
		Namespace(namespace).
		SubResource("exec").
		Param("container", container)

	req.VersionedParams(&coreV1.PodExecOptions{
		Container: container,
		Command:   []string{"bash", "-c", "cat > " + dstPath},
		Stdin:     true,
		Stdout:    true,
		Stderr:    true,
	}, scheme.ParameterCodec)

	exec, err := remotecommand.NewSPDYExecutor(config, "POST", req.URL())
	if err != nil {
		logrus.Errorf("error creating executor: %s", err)
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
		logrus.Errorf("error streaming: %s", err)
		return
	}

	logrus.Infof("file %s copied successfully", srcPath)
}
