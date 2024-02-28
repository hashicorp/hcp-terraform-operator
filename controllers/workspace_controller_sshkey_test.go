// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package controllers

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	tfc "github.com/hashicorp/go-tfe"
	appv1alpha2 "github.com/hashicorp/terraform-cloud-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName = newNamespacedName()
		workspace      = fmt.Sprintf("kubernetes-operator-%v", randomNumber())

		sshKeyName  = fmt.Sprintf("kubernetes-operator-sshkey-%v", randomNumber())
		sshKeyName2 = fmt.Sprintf("%v-2", sshKeyName)
		sshKeyID    = ""
		sshKeyID2   = ""
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)

		// Create an SSH keys
		sshKeyID = createSSHKey(sshKeyName)
		sshKeyID2 = createSSHKey(sshKeyName2)
	})

	AfterAll(func() {
		// Delete SSH keys
		err := tfClient.SSHKeys.Delete(ctx, sshKeyID)
		Expect(err).Should(Succeed())

		err = tfClient.SSHKeys.Delete(ctx, sshKeyID2)
		Expect(err).Should(Succeed())
	})

	BeforeEach(func() {
		// Create a new workspace object for each test
		instance = &appv1alpha2.Workspace{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "app.terraform.io/v1alpha2",
				Kind:       "Workspace",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:              namespacedName.Name,
				Namespace:         namespacedName.Namespace,
				DeletionTimestamp: nil,
				Finalizers:        []string{},
			},
			Spec: appv1alpha2.WorkspaceSpec{
				Organization: organization,
				Token: appv1alpha2.Token{
					SecretKeyRef: &corev1.SecretKeySelector{
						LocalObjectReference: corev1.LocalObjectReference{
							Name: secretNamespacedName.Name,
						},
						Key: secretKey,
					},
				},
				Name: workspace,
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		// Delete the Kubernetes workspace object and wait until the controller finishes the reconciliation after deletion of the object
		deleteWorkspace(instance)
	})

	Context("Workspace controller", func() {
		It("can handle SSH Key by name", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledSSHKeyByName(instance)

			// Delete the SSH key manually and wait until the controller revert this change
			ws, err := tfClient.Workspaces.UnassignSSHKey(ctx, instance.Status.WorkspaceID)
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			isReconciledSSHKeyByName(instance)

			// Update the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName2,
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledSSHKeyByName(instance)

			// Detach the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isSSHKeyEmpty(instance)
		})

		It("can handle SSH Key by id", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspace(instance)
			isReconciledSSHKeyByID(instance)

			// Delete the SSH key manually and wait until the controller revert this change
			ws, err := tfClient.Workspaces.UnassignSSHKey(ctx, instance.Status.WorkspaceID)
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			isReconciledSSHKeyByID(instance)

			// Update the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID2,
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledSSHKeyByID(instance)

			// Delete the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isSSHKeyEmpty(instance)
		})
	})
})

func isReconciledSSHKeyByName(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		if ws.SSHKey == nil {
			return false
		} else {
			s, err := tfClient.SSHKeys.Read(ctx, ws.SSHKey.ID)
			Expect(err).Should(Succeed())
			Expect(s).ShouldNot(BeNil())
			return s.Name == instance.Spec.SSHKey.Name
		}
	}).Should(BeTrue())
}

func isReconciledSSHKeyByID(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		if ws.SSHKey == nil {
			return false
		} else {
			return ws.SSHKey.ID == instance.Spec.SSHKey.ID
		}
	}).Should(BeTrue())
}

func isSSHKeyEmpty(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		return ws.SSHKey == nil && instance.Spec.SSHKey == nil
	}).Should(BeTrue())
}

func createSSHKey(sshKeyName string) string {
	sk, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(sk).ShouldNot(BeNil())
	Expect(err).Should(Succeed())
	var privateKeyBytes []byte = x509.MarshalPKCS1PrivateKey(sk)
	privateKeyBlock := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}
	privateSSHKey := pem.EncodeToMemory(privateKeyBlock)
	Expect(privateSSHKey).ShouldNot(BeNil())
	sshKey, err := tfClient.SSHKeys.Create(ctx, organization, tfc.SSHKeyCreateOptions{
		Name:  tfc.String(sshKeyName),
		Value: tfc.String(string(privateSSHKey)),
	})
	Expect(err).Should(Succeed())
	Expect(sshKey).ShouldNot(BeNil())

	return sshKey.ID
}
