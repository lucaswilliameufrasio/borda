package httpapi

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/lucaswilliameufrasio/borda/internal/hub"
	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

func TestShouldCreateRouterWithoutReturningThePassword(t *testing.T) {
	password := "plane-router-secret"
	passphrase := "plane-vault-secret"
	logs := &bytes.Buffer{}
	handler := testHandler(t, logs)
	body := `{
		"name": "Casa",
		"host": "connected.example",
		"username": "reader",
		"check_address": "192.0.2.1",
		"vault_kind": "file",
		"router_password": "` + password + `",
		"vault_passphrase": "` + passphrase + `"
	}`
	response := postJSON(handler, "/v1/routers", body)
	if response.Code != http.StatusCreated {
		t.Fatalf("status %d body %s", response.Code, response.Body.String())
	}
	assertNoSecret(t, response.Body.String(), password, passphrase)
	assertNoNull(t, response.Body.Bytes())
	if strings.Contains(response.Body.String(), "username") {
		t.Fatal("response includes the username")
	}
	if strings.Contains(logs.String(), password) || strings.Contains(logs.String(), passphrase) {
		t.Fatal("log includes a secret")
	}
	var created map[string]any
	if err := json.Unmarshal(response.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	id, ok := created["id"].(string)
	if !ok || id == "" {
		t.Fatal("missing id")
	}

	snapshot := get(handler, "/v1/routers/"+id+"/snapshot")
	if snapshot.Code != http.StatusOK {
		t.Fatalf("snapshot %d %s", snapshot.Code, snapshot.Body.String())
	}
	assertNoSecret(t, snapshot.Body.String(), password, passphrase)
	assertNoNull(t, snapshot.Body.Bytes())
	if !strings.Contains(snapshot.Body.String(), `"interface_name":"ether1"`) {
		t.Fatal("snapshot missing egress")
	}
	if !strings.Contains(snapshot.Body.String(), `"responding":true`) {
		t.Fatal("snapshot missing check")
	}

	other := postJSON(handler, "/v1/routers", `{
		"name": "Vazio",
		"host": "empty.example",
		"username": "reader",
		"vault_kind": "file",
		"router_password": "`+password+`",
		"vault_passphrase": "`+passphrase+`"
	}`)
	if other.Code != http.StatusCreated {
		t.Fatalf("second %d %s", other.Code, other.Body.String())
	}
	list := get(handler, "/v1/routers")
	if list.Code != http.StatusOK || !strings.Contains(list.Body.String(), "Casa") || !strings.Contains(list.Body.String(), "Vazio") {
		t.Fatalf("list %s", list.Body.String())
	}
	if strings.Contains(list.Body.String(), password) || strings.Contains(list.Body.String(), `"username"`) {
		t.Fatal("list leaked a credential field")
	}

	missing := get(handler, "/v1/routers/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/snapshot")
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), `"error_code":"ROUTER_NOT_FOUND"`) {
		t.Fatalf("missing %d %s", missing.Code, missing.Body.String())
	}
}

func TestShouldValidateMalformedAndUnknownFields(t *testing.T) {
	handler := testHandler(t, &bytes.Buffer{})
	invalid := postJSON(handler, "/v1/routers", `{"vault_kind":"file"}`)
	if invalid.Code != http.StatusUnprocessableEntity {
		t.Fatalf("status %d", invalid.Code)
	}
	var payload map[string]any
	if err := json.Unmarshal(invalid.Body.Bytes(), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["error_code"] != "INVALID_PARAMS" {
		t.Fatalf("code %v", payload["error_code"])
	}
	extra, ok := payload["extra"].(map[string]any)
	if !ok {
		t.Fatal("missing extra")
	}
	fields, ok := extra["validation_errors"].([]any)
	if !ok || len(fields) == 0 {
		t.Fatal("missing validation_errors")
	}

	malformed := postJSON(handler, "/v1/routers", `{`)
	if malformed.Code != http.StatusBadRequest || !strings.Contains(malformed.Body.String(), `"MALFORMED_REQUEST"`) {
		t.Fatalf("malformed %d %s", malformed.Code, malformed.Body.String())
	}
	unknown := postJSON(handler, "/v1/routers", `{"password":"nope"}`)
	if unknown.Code != http.StatusUnprocessableEntity || !strings.Contains(unknown.Body.String(), "desconhecido") {
		t.Fatalf("unknown %d %s", unknown.Code, unknown.Body.String())
	}
	assertNoNull(t, invalid.Body.Bytes())
}

func TestShouldAllowOnlyLocalhostCORS(t *testing.T) {
	handler := testHandler(t, &bytes.Buffer{})
	request := httptest.NewRequest(http.MethodOptions, "/v1/routers", nil)
	request.Header.Set("Origin", "http://127.0.0.1:5173")
	request.Header.Set("Access-Control-Request-Method", "POST")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	if response.Code != http.StatusNoContent {
		t.Fatalf("status %d", response.Code)
	}
	if response.Header().Get("Access-Control-Allow-Origin") != "http://127.0.0.1:5173" {
		t.Fatal("missing allow origin")
	}
	denied := httptest.NewRequest(http.MethodGet, "/healthz", nil)
	denied.Header.Set("Origin", "https://example.invalid")
	deniedResponse := httptest.NewRecorder()
	handler.ServeHTTP(deniedResponse, denied)
	if deniedResponse.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Fatal("reflected an unknown origin")
	}
	if deniedResponse.Code != http.StatusOK {
		t.Fatalf("health %d", deniedResponse.Code)
	}
}

func testHandler(t *testing.T, logs io.Writer) http.Handler {
	t.Helper()
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("connected.example", "identity\tExample\t7.0\nroute\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\nlease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tphone\ncheck\t1\n")
	memory.Set("empty.example", "identity\tEmpty\t7.0\n")
	store, err := hub.Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), memory)
	if err != nil {
		t.Fatal(err)
	}
	server := New(store, log.New(logs, "", 0), []string{"http://127.0.0.1:5173", "http://localhost:5173"})
	return server.Handler()
}

func postJSON(handler http.Handler, path string, body string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	request.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func get(handler http.Handler, path string) *httptest.ResponseRecorder {
	request := httptest.NewRequest(http.MethodGet, path, nil)
	response := httptest.NewRecorder()
	handler.ServeHTTP(response, request)
	return response
}

func assertNoSecret(t *testing.T, body string, secrets ...string) {
	t.Helper()
	for _, secretValue := range secrets {
		if strings.Contains(body, secretValue) {
			t.Fatal("response contains a secret")
		}
	}
}

func assertNoNull(t *testing.T, body []byte) {
	t.Helper()
	var value any
	if err := json.Unmarshal(body, &value); err != nil {
		t.Fatal(err)
	}
	walkJSON(t, value)
}

func walkJSON(t *testing.T, value any) {
	t.Helper()
	switch typed := value.(type) {
	case map[string]any:
		for _, item := range typed {
			if item == nil {
				t.Fatal("response contains null")
			}
			walkJSON(t, item)
		}
	case []any:
		for _, item := range typed {
			if item == nil {
				t.Fatal("response contains null")
			}
			walkJSON(t, item)
		}
	}
}
