package routeros

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestShouldKeepPasswordOutOfSSHArguments(t *testing.T) {
	dir := t.TempDir()
	password := "plane-router-secret"
	fake := filepath.Join(dir, "ssh")
	body := "#!/bin/sh\n" +
		"printf '%s\\n' \"$@\" > \"" + dir + "/args\"\n" +
		"if [ -n \"$SSH_ASKPASS\" ]; then cat \"$SSH_ASKPASS\" > \"" + dir + "/askpass\"; fi\n" +
		"printf '%s\\n' 'identity\tLab\t7.0'\n"
	if err := os.WriteFile(fake, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	reader := &SSHReader{Command: fake, KnownHostsPath: filepath.Join(dir, "known_hosts")}
	reading, err := reader.Read(context.Background(), Target{
		Host:     "connected.example",
		Port:     22,
		Username: "reader",
		Password: password,
	})
	if err != nil {
		t.Fatal(err)
	}
	if reading.IdentityName != "Lab" {
		t.Fatalf("identity %q", reading.IdentityName)
	}
	args, err := os.ReadFile(filepath.Join(dir, "args"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(args), password) {
		t.Fatal("password appeared in ssh arguments")
	}
	askpass, err := os.ReadFile(filepath.Join(dir, "askpass"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(askpass), password) {
		t.Fatal("password was written into the askpass program")
	}
	if !strings.Contains(string(args), "identity") {
		t.Fatal("remote command was not passed to ssh")
	}
	for _, word := range []string{" add ", " set ", " remove "} {
		if strings.Contains(string(args), word) {
			t.Fatalf("remote command contains %q", word)
		}
	}
}

func TestShouldMapSSHPermissionDenied(t *testing.T) {
	dir := t.TempDir()
	fake := filepath.Join(dir, "ssh")
	body := "#!/bin/sh\n" +
		"echo 'Permission denied, please try again.' >&2\n" +
		"exit 255\n"
	if err := os.WriteFile(fake, []byte(body), 0o700); err != nil {
		t.Fatal(err)
	}
	reader := &SSHReader{Command: fake, KnownHostsPath: filepath.Join(dir, "known_hosts")}
	_, err := reader.Read(context.Background(), Target{
		Host:     "connected.example",
		Port:     22,
		Username: "reader",
		Password: "plane-router-secret",
	})
	if !errorsIs(err, ErrAuthenticationRejected) {
		t.Fatalf("err %v", err)
	}
}

func errorsIs(err error, target error) bool {
	return err == target
}
