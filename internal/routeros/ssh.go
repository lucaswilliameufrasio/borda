package routeros

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// SSHReader runs the read-only script over OpenSSH.
// The password is passed to SSH_ASKPASS through the environment, never as an argument.
type SSHReader struct {
	Command        string
	KnownHostsPath string
}

func (reader *SSHReader) Read(ctx context.Context, target Target) (Reading, error) {
	script, err := ReadScript(target.CheckAddress)
	if err != nil {
		return Reading{}, err
	}
	stdout, err := reader.run(ctx, target, script)
	if err != nil {
		return Reading{}, err
	}
	return ParseReading(stdout), nil
}

// Prove disables the commented primary route, measures USB and LAN, then enables it again.
// The enable step uses a context that survives cancellation of the caller's context.
func (reader *SSHReader) Prove(ctx context.Context, target Target) (Proof, error) {
	reading, err := reader.Read(ctx, target)
	if err != nil {
		return Proof{}, err
	}
	plan, err := PlanProof(reading, target.CheckAddress)
	if err != nil {
		return Proof{}, err
	}
	if _, err := reader.run(ctx, target, DisableScript()); err != nil {
		return Proof{}, err
	}
	usbReceived, usbErr := reader.measure(ctx, target, target.CheckAddress, plan.USBInterface, "")
	lanReceived, lanErr := reader.measure(ctx, target, target.CheckAddress, "", plan.LANAddress)
	restoreCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 15*time.Second)
	defer cancel()
	if _, err := reader.run(restoreCtx, target, EnableScript()); err != nil {
		return Proof{Outcome: OutcomeFailed, PrimaryActive: false}, ErrProofNotRestored
	}
	verified, err := reader.Read(restoreCtx, target)
	if err != nil || !PrimaryIsActive(verified) {
		return Proof{Outcome: OutcomeFailed, PrimaryActive: false}, ErrProofNotRestored
	}
	if usbErr != nil {
		usbReceived = 0
	}
	if lanErr != nil {
		lanReceived = 0
	}
	outcome := OutcomePassed
	if usbReceived == 0 || lanReceived == 0 {
		outcome = OutcomeFailed
	}
	return Proof{
		Outcome:       outcome,
		PrimaryActive: true,
		USBReceived:   usbReceived,
		LANReceived:   lanReceived,
	}, nil
}

// RegisterWAN removes the port from the bridge, then creates the commented NAT and DHCP client.
// A later script failure returns ErrWanIncomplete so a repeat can finish the same comment.
func (reader *SSHReader) RegisterWAN(ctx context.Context, target Target, request WanRequest) (WanResult, error) {
	reading, err := reader.Read(ctx, target)
	if err != nil {
		return WanResult{}, err
	}
	plan, err := PlanWan(reading, request)
	if err != nil {
		return WanResult{}, err
	}
	result := WanResult{
		Comment:        plan.Comment,
		InterfaceName:  plan.InterfaceName,
		Distance:       plan.Distance,
		AlreadyPresent: plan.AlreadyPresent,
	}
	if plan.AlreadyPresent {
		return result, nil
	}
	scripts, err := WanScripts(plan)
	if err != nil {
		return WanResult{}, err
	}
	started := false
	for _, script := range scripts {
		if _, err := reader.run(ctx, target, script); err != nil {
			if started {
				return WanResult{}, ErrWanIncomplete
			}
			return WanResult{}, err
		}
		started = true
	}
	return result, nil
}

func (reader *SSHReader) measure(ctx context.Context, target Target, checkAddress string, iface string, source string) (int, error) {
	script, err := PingScript(checkAddress, iface, source)
	if err != nil {
		return 0, err
	}
	stdout, err := reader.run(ctx, target, script)
	if err != nil {
		return 0, err
	}
	return parseReceived(stdout)
}

func parseReceived(output string) (int, error) {
	for _, line := range strings.Split(output, "\n") {
		fields := strings.Split(strings.TrimSpace(line), "\t")
		if len(fields) >= 2 && fields[0] == "received" {
			received, err := strconv.Atoi(fields[1])
			if err != nil {
				return 0, err
			}
			return received, nil
		}
	}
	return 0, errors.New("missing measure")
}

func (reader *SSHReader) run(ctx context.Context, target Target, script string) (string, error) {
	if target.Password == "" {
		return "", ErrAuthenticationRejected
	}
	if !safeEndpoint(target.Host) || !safeEndpoint(target.Username) || target.Port < 1 || target.Port > 65535 {
		return "", ErrUnreachable
	}
	if reader.KnownHostsPath == "" {
		return "", ErrUnreachable
	}
	askpass, err := os.CreateTemp("", "borda-askpass-")
	if err != nil {
		return "", ErrUnreachable
	}
	askpassPath := askpass.Name()
	defer os.Remove(askpassPath)
	if _, err := askpass.WriteString("#!/bin/sh\nprintf '%s\\n' \"$BORDA_ROUTER_PASSWORD\"\n"); err != nil {
		askpass.Close()
		return "", ErrUnreachable
	}
	if err := askpass.Close(); err != nil {
		return "", ErrUnreachable
	}
	if err := os.Chmod(askpassPath, 0o700); err != nil {
		return "", ErrUnreachable
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
		return "", classifySSHError(err, stderr.String())
	}
	return stdout.String(), nil
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
