package routeros

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// SSHReader runs the read-only script over OpenSSH.
// The password is passed to SSH_ASKPASS through the environment, never as an argument.
type SSHReader struct {
	Command        string
	KnownHostsPath string
}

func (reader *SSHReader) Read(ctx context.Context, target Target) (Reading, error) {
	if target.Password == "" {
		return Reading{}, ErrAuthenticationRejected
	}
	if !safeEndpoint(target.Host) || !safeEndpoint(target.Username) || target.Port < 1 || target.Port > 65535 {
		return Reading{}, ErrUnreachable
	}
	script, err := ReadScript(target.CheckAddress)
	if err != nil {
		return Reading{}, err
	}
	if reader.KnownHostsPath == "" {
		return Reading{}, ErrUnreachable
	}
	askpass, err := os.CreateTemp("", "borda-askpass-")
	if err != nil {
		return Reading{}, ErrUnreachable
	}
	askpassPath := askpass.Name()
	defer os.Remove(askpassPath)
	if _, err := askpass.WriteString("#!/bin/sh\nprintf '%s\\n' \"$BORDA_ROUTER_PASSWORD\"\n"); err != nil {
		askpass.Close()
		return Reading{}, ErrUnreachable
	}
	if err := askpass.Close(); err != nil {
		return Reading{}, ErrUnreachable
	}
	if err := os.Chmod(askpassPath, 0o700); err != nil {
		return Reading{}, ErrUnreachable
	}

	commandName := reader.Command
	if commandName == "" {
		commandName = "ssh"
	}
	command := exec.CommandContext(
		ctx,
		commandName,
		"-F", "/dev/null",
		"-o", "IdentitiesOnly=yes",
		"-o", "PubkeyAuthentication=no",
		"-o", "PreferredAuthentications=password",
		"-o", "NumberOfPasswordPrompts=1",
		"-o", "StrictHostKeyChecking=accept-new",
		"-o", "UserKnownHostsFile="+reader.KnownHostsPath,
		"-o", "GlobalKnownHostsFile=/dev/null",
		"-o", "ConnectTimeout=8",
		"-p", fmt.Sprintf("%d", target.Port),
		"-l", target.Username,
		target.Host,
		script,
	)
	command.Env = []string{
		"PATH=" + os.Getenv("PATH"),
		"BORDA_ROUTER_PASSWORD=" + target.Password,
		"SSH_ASKPASS=" + askpassPath,
		"SSH_ASKPASS_REQUIRE=force",
		"DISPLAY=borda",
		"LANG=C",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	command.Stdout = &stdout
	command.Stderr = &stderr
	err = command.Run()
	if err != nil {
		return Reading{}, classifySSHError(err, stderr.String())
	}
	return ParseReading(stdout.String()), nil
}

func classifySSHError(err error, stderr string) error {
	lower := strings.ToLower(stderr)
	if strings.Contains(lower, "permission denied") || strings.Contains(lower, "authentication") {
		return ErrAuthenticationRejected
	}
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return ErrUnreachable
	}
	return ErrUnreachable
}

func safeEndpoint(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") {
		return false
	}
	return safeCheckAddress.MatchString(value)
}
