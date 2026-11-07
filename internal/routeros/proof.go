package routeros

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
)

var ErrProofNotReady = errors.New("proof not ready")
var ErrProofNotRestored = errors.New("proof not restored")

const (
	ReasonMissingCheck   = "missing_check"
	ReasonMissingPrimary = "missing_primary"
	ReasonMissingReserve = "missing_reserve"
	ReasonMissingForward = "missing_forward"
	ReasonMissingLAN     = "missing_lan"
)

var safeInterfaceName = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,64}$`)

// ProofError explains why a proof refused to disable the primary route.
type ProofError struct {
	Reason string
}

func (err *ProofError) Error() string {
	return "proof not ready"
}

func (err *ProofError) Unwrap() error {
	return ErrProofNotReady
}

// ProofPlan is the measured path. It is computed before any write.
type ProofPlan struct {
	USBInterface string
	LANAddress   string
}

// PlanProof decides whether the proof may start. It does not change the router.
func PlanProof(reading Reading, checkAddress string) (ProofPlan, error) {
	if checkAddress == "" || !CheckAddressSafe(checkAddress) {
		return ProofPlan{}, &ProofError{Reason: ReasonMissingCheck}
	}
	primaryCount := 0
	primaryInterface := ""
	for _, route := range reading.Routes {
		if route.Comment != PrimaryComment {
			continue
		}
		primaryCount = primaryCount + 1
		primaryInterface = route.InterfaceName
	}
	if primaryCount != 1 || !safeName(primaryInterface) {
		return ProofPlan{}, &ProofError{Reason: ReasonMissingPrimary}
	}
	reserve := ""
	reserveCount := 0
	for _, modem := range reading.LTE {
		if !modem.Running || !safeName(modem.Name) {
			continue
		}
		if !interfaceHasAddress(reading, modem.Name) {
			continue
		}
		reserveCount = reserveCount + 1
		reserve = modem.Name
	}
	if reserveCount != 1 {
		return ProofPlan{}, &ProofError{Reason: ReasonMissingReserve}
	}
	forwardReady := false
	for _, rule := range reading.Forward {
		if rule.Action == "accept" && rule.OutInterface == reserve {
			forwardReady = true
		}
	}
	if !forwardReady {
		return ProofPlan{}, &ProofError{Reason: ReasonMissingForward}
	}
	lanAddress := ""
	for _, address := range reading.Addresses {
		if address.InterfaceName == reserve || address.InterfaceName == primaryInterface {
			continue
		}
		host := hostAddress(address.Value)
		if host != "" && CheckAddressSafe(host) {
			lanAddress = host
			break
		}
	}
	if lanAddress == "" {
		return ProofPlan{}, &ProofError{Reason: ReasonMissingLAN}
	}
	return ProofPlan{USBInterface: reserve, LANAddress: lanAddress}, nil
}

// PrimaryIsActive reports whether the single commented primary route is active.
func PrimaryIsActive(reading Reading) bool {
	found := false
	active := false
	for _, route := range reading.Routes {
		if route.Comment != PrimaryComment {
			continue
		}
		if found {
			return false
		}
		found = true
		active = route.Active
	}
	return found && active
}

// DisableScript disables only the commented primary default route.
func DisableScript() string {
	return `/ip route disable [find comment="` + PrimaryComment + `" dst-address=0.0.0.0/0]`
}

// EnableScript enables only the commented primary default route.
func EnableScript() string {
	return `/ip route enable [find comment="` + PrimaryComment + `" dst-address=0.0.0.0/0]`
}

// PingScript measures one public address through an interface or from a LAN address.
// proplist=received is required on RouterOS 7 when count is 1.
func PingScript(checkAddress string, iface string, source string) (string, error) {
	if !CheckAddressSafe(checkAddress) {
		return "", fmt.Errorf("invalid check address")
	}
	if iface != "" && !safeName(iface) {
		return "", fmt.Errorf("invalid interface")
	}
	if source != "" && !CheckAddressSafe(source) {
		return "", fmt.Errorf("invalid source")
	}
	if iface == "" && source == "" {
		return "", fmt.Errorf("missing measure path")
	}
	qualifier := ""
	if iface != "" {
		qualifier = ` interface="` + iface + `"`
	}
	if source != "" {
		qualifier = ` src-address="` + source + `"`
	}
	return `:local result [/ping address="` + checkAddress + `"` + qualifier + ` count=1 proplist=received as-value]; :put ("received` + "\t" + `" . ($result->"received"))`, nil
}

func safeName(value string) bool {
	if value == "" || strings.HasPrefix(value, "-") {
		return false
	}
	return safeInterfaceName.MatchString(value)
}

func interfaceHasAddress(reading Reading, name string) bool {
	for _, address := range reading.Addresses {
		if address.InterfaceName == name && strings.TrimSpace(address.Value) != "" {
			return true
		}
	}
	return false
}

func hostAddress(value string) string {
	slash := strings.Index(value, "/")
	if slash >= 0 {
		return value[:slash]
	}
	return value
}
