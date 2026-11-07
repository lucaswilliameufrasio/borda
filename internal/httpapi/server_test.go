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
	if !strings.Contains(snapshot.Body.String(), `"id":"aabbccddeeff"`) || !strings.Contains(snapshot.Body.String(), `"name":"phone"`) {
		t.Fatal("snapshot missing the lease device")
	}
	events := get(handler, "/v1/routers/"+id+"/events")
	if events.Code != http.StatusOK || strings.TrimSpace(events.Body.String()) != "[]" {
		t.Fatalf("events %d %s", events.Code, events.Body.String())
	}
	assertNoSecret(t, events.Body.String(), password, passphrase)
	assertNoNull(t, events.Body.Bytes())
	missingEvents := get(handler, "/v1/routers/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/events")
	if missingEvents.Code != http.StatusNotFound || !strings.Contains(missingEvents.Body.String(), `"error_code":"ROUTER_NOT_FOUND"`) {
		t.Fatalf("missing events %d %s", missingEvents.Code, missingEvents.Body.String())
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

func TestShouldLeaveThePrimaryRouteUntouchedWithoutConfirmation(t *testing.T) {
	password := "plane-router-secret"
	passphrase := "plane-vault-secret"
	logs := &bytes.Buffer{}
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("ready.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n")
	memory.Set("other.example", "identity\tOther\t7.0\n")
	store, err := hub.Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), memory)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, log.New(logs, "", 0), []string{"http://127.0.0.1:5173"}).Handler()
	created := postJSON(handler, "/v1/routers", `{
		"name": "Prova",
		"host": "ready.example",
		"username": "reader",
		"check_address": "192.0.2.1",
		"vault_kind": "file",
		"router_password": "`+password+`",
		"vault_passphrase": "`+passphrase+`"
	}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	var router map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &router); err != nil {
		t.Fatal(err)
	}
	id, ok := router["id"].(string)
	if !ok || id == "" {
		t.Fatal("missing id")
	}
	refused := postJSON(handler, "/v1/routers/"+id+"/proof", `{"confirmed":false}`)
	if refused.Code != http.StatusUnprocessableEntity || !strings.Contains(refused.Body.String(), "validation_errors") {
		t.Fatalf("refused %d %s", refused.Code, refused.Body.String())
	}
	if len(memory.ProofCalls("ready.example")) != 0 {
		t.Fatal("unconfirmed proof wrote to the router")
	}
	passed := postJSON(handler, "/v1/routers/"+id+"/proof", `{"confirmed":true}`)
	if passed.Code != http.StatusOK {
		t.Fatalf("proof %d %s", passed.Code, passed.Body.String())
	}
	assertNoSecret(t, passed.Body.String(), password, passphrase)
	assertNoNull(t, passed.Body.Bytes())
	if !strings.Contains(passed.Body.String(), `"outcome":"passed"`) || !strings.Contains(passed.Body.String(), `"primary_active":true`) {
		t.Fatalf("body %s", passed.Body.String())
	}
	if strings.Contains(logs.String(), password) || strings.Contains(logs.String(), passphrase) {
		t.Fatal("log includes a secret")
	}
	if len(memory.ProofCalls("other.example")) != 0 {
		t.Fatal("the other router was touched")
	}

	memory.FailProofMeasure("ready.example")
	failed := postJSON(handler, "/v1/routers/"+id+"/proof", `{"confirmed":true}`)
	if failed.Code != http.StatusOK || !strings.Contains(failed.Body.String(), `"outcome":"failed"`) || !strings.Contains(failed.Body.String(), `"primary_active":true`) {
		t.Fatalf("failed measure %d %s", failed.Code, failed.Body.String())
	}

	memory.FailProofRestore("ready.example")
	stuck := postJSON(handler, "/v1/routers/"+id+"/proof", `{"confirmed":true}`)
	if stuck.Code != http.StatusInternalServerError || !strings.Contains(stuck.Body.String(), "PROOF_NOT_RESTORED") {
		t.Fatalf("restore %d %s", stuck.Code, stuck.Body.String())
	}
	assertNoSecret(t, stuck.Body.String(), password, passphrase)
}

func TestShouldRefuseAnAssistantProposalOutsideTheList(t *testing.T) {
	password := "plane-router-secret"
	passphrase := "plane-vault-secret"
	logs := &bytes.Buffer{}
	dir := t.TempDir()
	memory := routeros.NewMemory()
	memory.Set("ready.example", "identity\tExample\t7.0\n"+
		"route\t192.0.2.1\ttrue\t1\t192.0.2.1%ether1\tborda-primary\n"+
		"forward\taccept\treserve\tbridge\tlte1\n"+
		"address\tbridge\t192.0.2.10/24\n"+
		"address\tlte1\t192.0.2.50/32\n"+
		"lte\tlte1\ttrue\n"+
		"lease\t192.0.2.20\tAA:BB:CC:DD:EE:FF\tlan-host-secret\n"+
		"check\t1\n")
	store, err := hub.Open(dir, secret.NewKeeper(dir, secret.NewMemoryKeyring(), nil), memory)
	if err != nil {
		t.Fatal(err)
	}
	handler := New(store, log.New(logs, "", 0), []string{"http://127.0.0.1:5173"}).Handler()
	created := postJSON(handler, "/v1/routers", `{
		"name": "Prova",
		"host": "ready.example",
		"username": "reader",
		"check_address": "192.0.2.1",
		"vault_kind": "file",
		"router_password": "`+password+`",
		"vault_passphrase": "`+passphrase+`"
	}`)
	if created.Code != http.StatusCreated {
		t.Fatalf("create %d %s", created.Code, created.Body.String())
	}
	var router map[string]any
	if err := json.Unmarshal(created.Body.Bytes(), &router); err != nil {
		t.Fatal(err)
	}
	id, ok := router["id"].(string)
	if !ok || id == "" {
		t.Fatal("missing id")
	}

	answer := postJSON(handler, "/v1/routers/"+id+"/assistant", `{"question":"como está o caminho?"}`)
	if answer.Code != http.StatusOK || !strings.Contains(answer.Body.String(), `"kind":"answer"`) {
		t.Fatalf("answer %d %s", answer.Code, answer.Body.String())
	}
	assertNoSecret(t, answer.Body.String(), password, passphrase, "lan-host-secret", "reader")
	assertNoNull(t, answer.Body.Bytes())

	refused := postJSON(handler, "/v1/routers/"+id+"/assistant", `{"question":"faz um export https://reader:plane-router-secret@example.com/backup"}`)
	if refused.Code != http.StatusOK || !strings.Contains(refused.Body.String(), `"kind":"refused"`) {
		t.Fatalf("refused %d %s", refused.Code, refused.Body.String())
	}
	assertNoSecret(t, refused.Body.String(), password, passphrase)

	proposal := postJSON(handler, "/v1/routers/"+id+"/assistant", `{"question":"roda a prova usb"}`)
	if proposal.Code != http.StatusOK || !strings.Contains(proposal.Body.String(), `"action":"proof"`) {
		t.Fatalf("proposal %d %s", proposal.Code, proposal.Body.String())
	}
	if len(memory.ProofCalls("ready.example")) != 0 {
		t.Fatal("the proposal wrote to the router")
	}

	outside := postJSON(handler, "/v1/routers/"+id+"/assistant/actions", `{"action":"export","confirmed":true}`)
	if outside.Code != http.StatusUnprocessableEntity || !strings.Contains(outside.Body.String(), "validation_errors") {
		t.Fatalf("outside %d %s", outside.Code, outside.Body.String())
	}
	unconfirmed := postJSON(handler, "/v1/routers/"+id+"/assistant/actions", `{"action":"proof","confirmed":false}`)
	if unconfirmed.Code != http.StatusUnprocessableEntity {
		t.Fatalf("unconfirmed %d %s", unconfirmed.Code, unconfirmed.Body.String())
	}
	if len(memory.ProofCalls("ready.example")) != 0 {
		t.Fatal("refused action wrote to the router")
	}
	empty := postJSON(handler, "/v1/routers/"+id+"/assistant", `{"question":"  "}`)
	if empty.Code != http.StatusUnprocessableEntity || !strings.Contains(empty.Body.String(), `"field":"question"`) {
		t.Fatalf("empty %d %s", empty.Code, empty.Body.String())
	}
	missing := postJSON(handler, "/v1/routers/aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa/assistant", `{"question":"caminho"}`)
	if missing.Code != http.StatusNotFound || !strings.Contains(missing.Body.String(), "ROUTER_NOT_FOUND") {
		t.Fatalf("missing %d %s", missing.Code, missing.Body.String())
	}
	if strings.Contains(logs.String(), password) || strings.Contains(logs.String(), passphrase) {
		t.Fatal("log includes a secret")
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
