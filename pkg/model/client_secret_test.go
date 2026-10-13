package model

import (
	"testing"

	"github.com/keycloak/keycloak-realm-operator/pkg/apis/keycloak/v1alpha1"
	v1 "k8s.io/api/core/v1"
	v12 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/stretchr/testify/assert"
)

func newSecretTestCR() *v1alpha1.KeycloakClient {
	cr := &v1alpha1.KeycloakClient{}
	cr.Name = "hub-ui"
	cr.Namespace = "hub"
	cr.Spec.Client = &v1alpha1.KeycloakAPIClient{ClientID: "hub-ui", Secret: "s3cret"}
	return cr
}

func TestClientSecret_IsExcludedFromBackup(t *testing.T) {
	secret := ClientSecret(newSecretTestCR())

	assert.Equal(t, "true", secret.Labels[veleroExcludeFromBackupLabel])
}

func TestClientSecretReconciled_AddsExclusionLabelToUnlabelledSecret(t *testing.T) {
	current := &v1.Secret{ObjectMeta: v12.ObjectMeta{Labels: map[string]string{"app": ApplicationName}}}

	reconciled := ClientSecretReconciled(newSecretTestCR(), current)

	assert.Equal(t, "true", reconciled.Labels[veleroExcludeFromBackupLabel])
	assert.Equal(t, ApplicationName, reconciled.Labels["app"])
	assert.NotContains(t, current.Labels, veleroExcludeFromBackupLabel)
}

func TestClientSecretReconciled_HandlesNilLabels(t *testing.T) {
	reconciled := ClientSecretReconciled(newSecretTestCR(), &v1.Secret{})

	assert.Equal(t, "true", reconciled.Labels[veleroExcludeFromBackupLabel])
}
