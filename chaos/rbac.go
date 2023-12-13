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
	coreV1 "k8s.io/api/core/v1"
	rbacV1 "k8s.io/api/rbac/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func addServiceAccount(namespace string) error {
	_, err := client.CoreV1().ServiceAccounts(namespace).Create(context.TODO(), &coreV1.ServiceAccount{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "chaos-admin",
			Labels: map[string]string{
				"name": "chaos-admin",
			},
		},
	}, metaV1.CreateOptions{})
	if err.(*errors.StatusError).ErrStatus.Reason != metaV1.StatusReasonAlreadyExists {
		return err
	} else {
		return nil
	}
}

func addRoleBinding(namespace string) error {
	_, err := client.RbacV1().ClusterRoleBindings().Create(context.TODO(), &rbacV1.ClusterRoleBinding{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "chaos-admin",
			Labels: map[string]string{
				"name": "chaos-admin",
			},
		},
		Subjects: []rbacV1.Subject{
			{
				Kind:      rbacV1.ServiceAccountKind,
				Name:      "chaos-admin",
				Namespace: namespace,
			},
		},
		RoleRef: rbacV1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "chaos-admin",
		},
	}, metaV1.CreateOptions{})
	if err.(*errors.StatusError).ErrStatus.Reason != metaV1.StatusReasonAlreadyExists {
		return err
	} else {
		return nil
	}
}

func addClusterRole() error {
	_, err := client.RbacV1().ClusterRoles().Create(context.TODO(), &rbacV1.ClusterRole{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "chaos-admin",
			Labels: map[string]string{
				"name": "chaos-admin",
			},
		},
		Rules: []rbacV1.PolicyRule{
			{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			{
				Verbs:           []string{"*"},
				NonResourceURLs: []string{"*"},
			},
		},
	}, metaV1.CreateOptions{})
	if err.(*errors.StatusError).ErrStatus.Reason != metaV1.StatusReasonAlreadyExists {
		return err
	} else {
		return nil
	}
}
