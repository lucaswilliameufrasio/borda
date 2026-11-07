package routeros

import (
	"context"
	"errors"
	"regexp"
)

var ErrUnreachable = errors.New("router unreachable")
var ErrAuthenticationRejected = errors.New("authentication rejected")

var safeCheckAddress = regexp.MustCompile(`^[A-Za-z0-9.:-]{1,253}$`)

// CheckAddressSafe reports whether a tenant check address can be embedded in a read script.
func CheckAddressSafe(value string) bool {
	return safeCheckAddress.MatchString(value)
}

// Target is one read-only session.
type Target struct {
	Host         string
	Port         int
	Username     string
	Password     string
	CheckAddress string
}

// Route is one default route.
type Route struct {
	Gateway       string
	Active        bool
	Distance      int
	InterfaceName string
	Comment       string
}

// ForwardRule is one forward filter rule.
type ForwardRule struct {
	Action       string
	Comment      string
	InInterface  string
	OutInterface string
}

// Lease is one DHCP lease.
type Lease struct {
	Address  string
	MAC      string
	HostName string
}

// Address is one assigned address.
type Address struct {
	InterfaceName string
	Value         string
}

// LTE is one USB modem interface.
type LTE struct {
	Name    string
	Running bool
}

// Reading is the parsed read-only output.
type Reading struct {
	IdentityName    string
	IdentityVersion string
	Routes          []Route
	Forward         []ForwardRule
	Leases          []Lease
	ARP             []ARPEntry
	Neighbors       []Neighbor
	Addresses       []Address
	LTE             []LTE
	DHCPClients     []DHCPClient
	NAT             []NATRule
	BridgePorts     []BridgePort
	CheckReceived   *int
}

// PrimaryComment is the only route comment the USB proof may disable.
const PrimaryComment = "borda-primary"

// Proof is the result of one USB proof after the primary route is restored,
// or the state observed when restoration itself fails.
type Proof struct {
	Outcome       string
	PrimaryActive bool
	USBReceived   int
	LANReceived   int
}

const (
	OutcomePassed = "passed"
	OutcomeFailed = "failed"
)

// Reader performs named router operations.
// Read does not change the router. Prove may disable only the commented primary route,
// and it enables that route again before it returns.
// RegisterWAN creates only the DHCP client and NAT rule commented borda-wan-<name>.
type Reader interface {
	Read(ctx context.Context, target Target) (Reading, error)
	Prove(ctx context.Context, target Target) (Proof, error)
	RegisterWAN(ctx context.Context, target Target, request WanRequest) (WanResult, error)
}
