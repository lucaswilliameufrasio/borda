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
	Addresses       []Address
	LTE             []LTE
	CheckReceived   *int
}

// Reader performs named read operations. It does not change the router.
type Reader interface {
	Read(ctx context.Context, target Target) (Reading, error)
}
