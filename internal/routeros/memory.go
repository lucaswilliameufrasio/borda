package routeros

import (
	"context"
	"strings"
	"sync"
)

// Memory is a fake router keyed by host.
type Memory struct {
	mu          sync.Mutex
	hosts       map[string]string
	fail        map[string]error
	calls       map[string][]string
	disabled    map[string]bool
	failMeasure map[string]bool
	failRestore map[string]bool
}

func NewMemory() *Memory {
	return &Memory{
		hosts:       map[string]string{},
		fail:        map[string]error{},
		calls:       map[string][]string{},
		disabled:    map[string]bool{},
		failMeasure: map[string]bool{},
		failRestore: map[string]bool{},
	}
}

func (reader *Memory) Set(host string, output string) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.hosts[host] = output
}

func (reader *Memory) Fail(host string, err error) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.fail[host] = err
}

// FailProofMeasure makes the USB and LAN measures silent. The primary route is still restored.
func (reader *Memory) FailProofMeasure(host string) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.failMeasure[host] = true
}

// FailProofRestore leaves the commented primary route disabled after the enable attempt.
func (reader *Memory) FailProofRestore(host string) {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.failRestore[host] = true
}

// ProofCalls returns the write steps recorded for one host.
func (reader *Memory) ProofCalls(host string) []string {
	reader.mu.Lock()
	defer reader.mu.Unlock()
	calls := reader.calls[host]
	copied := make([]string, len(calls))
	copy(copied, calls)
	return copied
}

func (reader *Memory) Read(ctx context.Context, target Target) (Reading, error) {
	if err := ctx.Err(); err != nil {
		return Reading{}, err
	}
	if target.Password == "" {
		return Reading{}, ErrAuthenticationRejected
	}
	reader.mu.Lock()
	failure := reader.fail[target.Host]
	output := reader.hosts[target.Host]
	if reader.disabled[target.Host] {
		output = markPrimaryInactive(output)
	}
	reader.mu.Unlock()
	if failure != nil {
		return Reading{}, failure
	}
	return ParseReading(output), nil
}

// Prove runs the USB proof against the stored reading.
// It records disable and enable, and it does not write when the plan refuses.
func (reader *Memory) Prove(ctx context.Context, target Target) (Proof, error) {
	reading, err := reader.Read(ctx, target)
	if err != nil {
		return Proof{}, err
	}
	if _, err := PlanProof(reading, target.CheckAddress); err != nil {
		return Proof{}, err
	}
	reader.mu.Lock()
	defer reader.mu.Unlock()
	reader.calls[target.Host] = append(reader.calls[target.Host], "disable")
	reader.calls[target.Host] = append(reader.calls[target.Host], "enable")
	if reader.failRestore[target.Host] {
		reader.disabled[target.Host] = true
		return Proof{Outcome: OutcomeFailed, PrimaryActive: false}, ErrProofNotRestored
	}
	reader.disabled[target.Host] = false
	usbReceived := 1
	lanReceived := 1
	if reader.failMeasure[target.Host] {
		usbReceived = 0
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

func markPrimaryInactive(output string) string {
	lines := strings.Split(output, "\n")
	for index, line := range lines {
		fields := strings.Split(line, "\t")
		if len(fields) >= 6 && fields[0] == "route" && fields[5] == PrimaryComment {
			fields[2] = "false"
			lines[index] = strings.Join(fields, "\t")
		}
	}
	return strings.Join(lines, "\n")
}
