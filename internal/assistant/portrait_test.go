package assistant

import (
	"strings"
	"testing"
)

func TestShouldDropSecretsFromThePrompt(t *testing.T) {
	raw := []byte(`{
		"session": "connected",
		"identity_name": "lab",
		"has_active_route": true,
		"active_gateway": "192.0.2.1",
		"active_interface": "ether1",
		"active_distance": 1,
		"has_check": true,
		"check_address": "192.0.2.1",
		"check_responding": true,
		"interfaces": ["ether1"],
		"comments": ["borda-primary"],
		"distances": [1],
		"password": "vault-phrase-alpha",
		"username": "login-name-beta",
		"export": "export-body-gamma",
		"backup": "backup-blob-delta",
		"script": "script-body-epsilon",
		"credential_url": "https://login-name-beta:vault-phrase-alpha@example.com/secret",
		"host_name": "device-name-zeta",
		"mac": "mac-secret-eta",
		"user_content": "user-file-theta"
	}`)
	portrait, err := PortraitFromLoose(raw)
	if err != nil {
		t.Fatal(err)
	}
	question := "olha https://login-name-beta:vault-phrase-alpha@example.com/secret e me diz o caminho"
	body, err := Prompt(portrait, question)
	if err != nil {
		t.Fatal(err)
	}
	text := string(body)
	forbidden := []string{
		"vault-phrase-alpha",
		"login-name-beta",
		"export-body-gamma",
		"backup-blob-delta",
		"script-body-epsilon",
		"device-name-zeta",
		"mac-secret-eta",
		"user-file-theta",
	}
	for _, value := range forbidden {
		if strings.Contains(text, value) {
			t.Fatalf("prompt kept %s", value)
		}
	}
	if !strings.Contains(text, "192.0.2.1") || !strings.Contains(text, "borda-primary") {
		t.Fatalf("prompt dropped the path: %s", text)
	}
	if !strings.Contains(text, `"interfaces":["ether1"]`) {
		t.Fatalf("prompt dropped interfaces: %s", text)
	}
}
