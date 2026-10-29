package common

import (
	"context"

	"github.com/keycloak/keycloak-realm-operator/pkg/apis/keycloak/v1alpha1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// These kinds are not provided by the openshift api
const (
	RouteKind               = "Route"
	SecretKind              = "Secret"
	PodDisruptionBudgetKind = "PodDisruptionBudget"
	OpenShiftAPIServerKind  = "OpenShiftAPIServer"
)

// Try to get a list of keycloak instances that match the selector specified on the realm
func GetMatchingKeycloaks(ctx context.Context, c client.Client, labelSelector *v1.LabelSelector) (v1alpha1.ExternalKeycloakList, error) {
	var list v1alpha1.ExternalKeycloakList
	opts := []client.ListOption{
		client.MatchingLabels(labelSelector.MatchLabels),
	}

	err := c.List(ctx, &list, opts...)
	return list, err
}

// Try to get a list of keycloak instances that match the selector specified on the realm
func GetMatchingRealms(ctx context.Context, c client.Client, labelSelector *v1.LabelSelector) (v1alpha1.KeycloakRealmList, error) {
	var list v1alpha1.KeycloakRealmList
	opts := []client.ListOption{
		client.MatchingLabels(labelSelector.MatchLabels),
	}

	err := c.List(ctx, &list, opts...)
	return list, err
}
