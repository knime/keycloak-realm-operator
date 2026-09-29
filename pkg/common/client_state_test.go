package common

import (
	"context"
	"testing"

	kc "github.com/keycloak/keycloak-realm-operator/pkg/apis/keycloak/v1alpha1"
	"github.com/stretchr/testify/assert"
	apiErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// stubKeycloak implements only the calls made by ClientState.Read; any other call panics on the nil embedded interface.
type stubKeycloak struct {
	KeycloakInterface
	byID             *kc.KeycloakAPIClient
	byClientID       *kc.KeycloakAPIClient
	getClientCalls   int
	byClientIDCalls  int
	byClientIDLookup string
}

func (s *stubKeycloak) GetClient(clientID, realmName string) (*kc.KeycloakAPIClient, error) {
	s.getClientCalls++
	return s.byID, nil
}

func (s *stubKeycloak) GetClientByClientID(clientID, realmName string) (*kc.KeycloakAPIClient, error) {
	s.byClientIDCalls++
	s.byClientIDLookup = clientID
	return s.byClientID, nil
}

func (s *stubKeycloak) GetClientSecret(clientID, realmName string) (string, error) {
	return "keycloak-secret", nil
}

func (s *stubKeycloak) ListClientRoles(clientID, realmName string) ([]kc.RoleRepresentation, error) {
	return nil, nil
}

func (s *stubKeycloak) ListScopeMappings(clientID, realmName string) (*kc.MappingsRepresentation, error) {
	return &kc.MappingsRepresentation{}, nil
}

func (s *stubKeycloak) ListAvailableClientScopes(realmName string) ([]kc.KeycloakClientScope, error) {
	return nil, nil
}

func (s *stubKeycloak) ListDefaultClientScopes(clientID, realmName string) ([]kc.KeycloakClientScope, error) {
	return nil, nil
}

func (s *stubKeycloak) ListOptionalClientScopes(clientID, realmName string) ([]kc.KeycloakClientScope, error) {
	return nil, nil
}

func (s *stubKeycloak) GetRealm(realmName string) (*kc.KeycloakRealm, error) {
	return &kc.KeycloakRealm{Spec: kc.KeycloakRealmSpec{Realm: &kc.KeycloakAPIRealm{
		Realm:       realmName,
		DefaultRole: &kc.RoleRepresentation{ID: "default-role"},
	}}}, nil
}

func (s *stubKeycloak) ListRealmRoleClientRoleComposites(realmName, roleID, clientID string) ([]kc.RoleRepresentation, error) {
	return nil, nil
}

// notFoundClient reports every secret as absent, as after a restore that excludes client secrets.
type notFoundClient struct {
	client.Client
}

func (notFoundClient) Get(ctx context.Context, key client.ObjectKey, obj runtime.Object) error {
	return apiErrors.NewNotFound(schema.GroupResource{Resource: "secrets"}, key.Name)
}

func (notFoundClient) Update(context.Context, runtime.Object, ...client.UpdateOption) error {
	return nil
}

type recordingClient struct {
	notFoundClient
	updated *kc.KeycloakClient
}

func (c *recordingClient) Update(_ context.Context, obj runtime.Object, _ ...client.UpdateOption) error {
	c.updated = obj.(*kc.KeycloakClient).DeepCopy()
	return nil
}

func newClientCR(id string) *kc.KeycloakClient {
	cr := &kc.KeycloakClient{}
	cr.Name = "hub-ui"
	cr.Namespace = "hub"
	cr.Spec.Client = &kc.KeycloakAPIClient{ID: id, ClientID: "hub-ui"}
	return cr
}

func newRealmState() *ClientState {
	realm := &kc.KeycloakRealm{Spec: kc.KeycloakRealmSpec{Realm: &kc.KeycloakAPIRealm{Realm: "knime"}}}
	return NewClientState(context.TODO(), realm, kc.ExternalKeycloak{})
}

func TestClientState_Read_AdoptsExistingClientByClientIDWhenIDAbsent(t *testing.T) {
	// given a CR restored without an internal ID whose client already exists in Keycloak
	stub := &stubKeycloak{byClientID: &kc.KeycloakAPIClient{ID: "existing-uuid", ClientID: "hub-ui"}}
	cr := newClientCR("")
	state := newRealmState()
	controllerClient := &recordingClient{}

	// when
	err := state.Read(context.TODO(), cr, stub, controllerClient)

	// then the existing client is adopted instead of being reported as missing (which would trigger a duplicate create)
	assert.NoError(t, err)
	assert.NotNil(t, state.Client)
	assert.Equal(t, "existing-uuid", cr.Spec.Client.ID)
	assert.NotNil(t, controllerClient.updated)
	assert.Equal(t, "existing-uuid", controllerClient.updated.Spec.Client.ID)
	assert.Equal(t, "hub-ui", stub.byClientIDLookup)
	assert.Equal(t, 0, stub.getClientCalls)
}

func TestClientState_Read_ClientAbsentInKeycloakIsLeftForCreate(t *testing.T) {
	// given no client with this clientId exists
	stub := &stubKeycloak{}
	cr := newClientCR("")
	state := newRealmState()

	// when
	err := state.Read(context.TODO(), cr, stub, notFoundClient{})

	// then the reconciler sees no client and creates it
	assert.NoError(t, err)
	assert.Nil(t, state.Client)
	assert.Equal(t, "", cr.Spec.Client.ID)
}

func TestClientState_Read_KnownIDIsNotLookedUpByClientID(t *testing.T) {
	// given a CR whose ID still resolves in Keycloak
	stub := &stubKeycloak{byID: &kc.KeycloakAPIClient{ID: "known-uuid", ClientID: "hub-ui"}}
	cr := newClientCR("known-uuid")
	state := newRealmState()

	// when
	err := state.Read(context.TODO(), cr, stub, notFoundClient{})

	// then
	assert.NoError(t, err)
	assert.NotNil(t, state.Client)
	assert.Equal(t, "known-uuid", cr.Spec.Client.ID)
	assert.Equal(t, 0, stub.byClientIDCalls)
}

func TestClientState_Read_StaleIDFallsBackToClientID(t *testing.T) {
	// given a restored CR carrying an ID that no longer exists, but the clientId does
	stub := &stubKeycloak{byClientID: &kc.KeycloakAPIClient{ID: "new-uuid", ClientID: "hub-ui"}}
	cr := newClientCR("stale-uuid")
	state := newRealmState()

	// when
	err := state.Read(context.TODO(), cr, stub, notFoundClient{})

	// then the stale ID is replaced by the live one
	assert.NoError(t, err)
	assert.Equal(t, "new-uuid", cr.Spec.Client.ID)
}
