package common

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func newClientByClientIDServer(t *testing.T, body string) (*Client, func()) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		assert.Equal(t, "/auth/admin/realms/knime/clients", req.URL.Path)
		assert.Equal(t, "hub-ui", req.URL.Query().Get("clientId"))
		assert.Equal(t, http.MethodGet, req.Method)
		_, err := w.Write([]byte(body))
		assert.NoError(t, err)
	})
	server := httptest.NewServer(handler)
	return &Client{requester: server.Client(), URL: server.URL, token: "dummy"}, server.Close
}

func TestClient_GetClientByClientID_ReturnsExactMatch(t *testing.T) {
	c, closeServer := newClientByClientIDServer(t, `[{"id":"uuid-1","clientId":"hub-ui"}]`)
	defer closeServer()

	got, err := c.GetClientByClientID("hub-ui", "knime")

	assert.NoError(t, err)
	if assert.NotNil(t, got) {
		assert.Equal(t, "uuid-1", got.ID)
	}
}

func TestClient_GetClientByClientID_NoResultIsNil(t *testing.T) {
	c, closeServer := newClientByClientIDServer(t, `[]`)
	defer closeServer()

	got, err := c.GetClientByClientID("hub-ui", "knime")

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestClient_GetClientByClientID_MalformedResponseIsError(t *testing.T) {
	c, closeServer := newClientByClientIDServer(t, `<html>gateway error</html>`)
	defer closeServer()

	got, err := c.GetClientByClientID("hub-ui", "knime")

	// must not be reported as "client absent", which would trigger a duplicate create
	assert.Error(t, err)
	assert.Nil(t, got)
}

func TestClient_GetClientByClientID_IgnoresNonExactMatches(t *testing.T) {
	c, closeServer := newClientByClientIDServer(t, `[{"id":"uuid-2","clientId":"hub-ui-other"}]`)
	defer closeServer()

	got, err := c.GetClientByClientID("hub-ui", "knime")

	assert.NoError(t, err)
	assert.Nil(t, got)
}

func TestClient_GetClientByClientID_DuplicateIsError(t *testing.T) {
	c, closeServer := newClientByClientIDServer(t, `[{"id":"uuid-1","clientId":"hub-ui"},{"id":"uuid-2","clientId":"hub-ui"}]`)
	defer closeServer()

	got, err := c.GetClientByClientID("hub-ui", "knime")

	assert.Error(t, err)
	assert.Nil(t, got)
}
