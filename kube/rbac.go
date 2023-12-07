package kube

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
