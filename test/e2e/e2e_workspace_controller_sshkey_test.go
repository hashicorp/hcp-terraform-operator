// Copyright IBM Corp. 2022, 2025
// SPDX-License-Identifier: MPL-2.0

package controller

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"time"

	tfc "github.com/hashicorp/go-tfe"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	appv1alpha2 "github.com/hashicorp/hcp-terraform-operator/api/v1alpha2"
)

var _ = Describe("Workspace controller", Ordered, func() {
	var (
		instance       *appv1alpha2.Workspace
		namespacedName types.NamespacedName
		workspace      string

		sshKeyName  string
		sshKeyName2 string
		sshKeyID    string
		sshKeyID2   string
	)

	BeforeAll(func() {
		// Set default Eventually timers
		SetDefaultEventuallyTimeout(syncPeriod * 4)
		SetDefaultEventuallyPollingInterval(2 * time.Second)
	})

	BeforeEach(func() {
		namespacedName = newNamespacedName()
		workspace = fmt.Sprintf("kubernetes-operator-%v", randomNumber())
		sshKeyName = fmt.Sprintf("kubernetes-operator-sshkey-%v", randomNumber())
		sshKeyName2 = fmt.Sprintf("%v-2", sshKeyName)
		sshKeyID = createSSHKey(sshKeyName)
		sshKeyID2 = createSSHKey(sshKeyName2)
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
				Name:        workspace,
				Description: "SSH Key",
			},
			Status: appv1alpha2.WorkspaceStatus{},
		}
	})

	AfterEach(func() {
		deleteWorkspace(instance)
		Expect(tfClient.SSHKeys.Delete(ctx, sshKeyID)).Should(Succeed())
		Expect(tfClient.SSHKeys.Delete(ctx, sshKeyID2)).Should(Succeed())
	})

	Context("SSH Key", func() {
		It("can be created by name", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
		})
		It("can be created by ID", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
		})

		It("can be restored by name", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Delete the SSH key manually and wait until the controller revert this change
			ws, err := tfClient.Workspaces.UnassignSSHKey(ctx, instance.Status.WorkspaceID)
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			isReconciledSSHKey(instance)
		})
		It("can be restored by ID", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Delete the SSH key manually and wait until the controller revert this change
			ws, err := tfClient.Workspaces.UnassignSSHKey(ctx, instance.Status.WorkspaceID)
			Expect(err).Should(Succeed())
			Expect(ws).ShouldNot(BeNil())
			isReconciledSSHKey(instance)
		})

		It("can be updated by name", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Update the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName2,
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledSSHKey(instance)
		})
		It("can be updated by ID", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Update the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID2,
			}
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isReconciledSSHKey(instance)
		})

		It("can be removed by name", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				Name: sshKeyName,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Detach the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isSSHKeyEmpty(instance)
		})
		It("can be removed by ID", func() {
			instance.Spec.SSHKey = &appv1alpha2.SSHKey{
				ID: sshKeyID,
			}
			// Create a new Kubernetes workspace object and wait until the controller finishes the reconciliation
			createWorkspaceResource(instance)
			isReconciledSSHKey(instance)
			// Delete the SSH key
			Expect(k8sClient.Get(ctx, namespacedName, instance))
			instance.Spec.SSHKey = nil
			Expect(k8sClient.Update(ctx, instance)).Should(Succeed())
			isSSHKeyEmpty(instance)
		})
	})
})

func isReconciledSSHKey(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		if instance.Status.SSHKeyID == "" {
			return false
		}
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		if ws.SSHKey == nil {
			return false
		}
		return ws.SSHKey.ID == instance.Status.SSHKeyID
	}).Should(BeTrue())
}

func isSSHKeyEmpty(instance *appv1alpha2.Workspace) {
	namespacedName := getNamespacedName(instance)

	Eventually(func() bool {
		Expect(k8sClient.Get(ctx, namespacedName, instance)).Should(Succeed())
		if instance.Status.SSHKeyID != "" {
			return false
		}
		ws, err := tfClient.Workspaces.ReadByID(ctx, instance.Status.WorkspaceID)
		Expect(err).Should(Succeed())
		Expect(ws).ShouldNot(BeNil())
		return ws.SSHKey == nil && instance.Spec.SSHKey == nil
	}).Should(BeTrue())
}

func createSSHKey(sshKeyName string) string {
	sk, err := rsa.GenerateKey(rand.Reader, 2048)
	Expect(err).Should(Succeed())
	Expect(sk).ShouldNot(BeNil())
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
