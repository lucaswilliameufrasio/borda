package routeros

import (
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

// WanCommentPrefix marks the DHCP client and the NAT rule of one registered WAN.
// The USB proof never disables a route with this prefix. It only disables borda-primary.
const WanCommentPrefix = "borda-wan-"

const (
	ReasonNotSpare           = "not_spare"
	ReasonDistanceTaken      = "distance_taken"
	ReasonCommentTaken       = "comment_taken"
	ReasonMissingActiveRoute = "missing_active_route"
)

var ErrWanIncomplete = errors.New("wan registration incomplete")

var wanNamePattern = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9-]{0,23}$`)

// WanError explains why a WAN registration refused to write.
type WanError struct {
	Reason string
}

func (err *WanError) Error() string {
	return "wan not ready"
}

// DHCPClient is one DHCP client already on the router.
type DHCPClient struct {
	InterfaceName string
	Distance      int
	Comment       string
}

// NATRule is one source NAT rule.
type NATRule struct {
	Chain         string
	Action        string
	InterfaceName string
	Comment       string
}

// BridgePort is one interface sitting in a bridge.
type BridgePort struct {
	InterfaceName string
	Bridge        string
}

// WanRequest is the confirmed registration. The caller has already checked confirmation.
type WanRequest struct {
	Name          string
	InterfaceName string
	Distance      int
}

// WanResult is the stable comment that now owns the DHCP client and the NAT rule.
type WanResult struct {
	Comment        string
	InterfaceName  string
	Distance       int
	AlreadyPresent bool
}

// WanPlan is the write that RegisterWAN will run. AlreadyPresent means the scripts are skipped.
type WanPlan struct {
	Comment        string
	InterfaceName  string
	Distance       int
	AlreadyPresent bool
}

// WanComment returns the stable comment for a WAN name.
func WanComment(name string) (string, bool) {
	if !wanNamePattern.MatchString(name) {
		return "", false
	}
	return WanCommentPrefix + strings.ToLower(name), true
}

// InterfaceSafe reports whether an interface name can be embedded in a script.
func InterfaceSafe(value string) bool {
	return safeName(value)
}

// PlanWan decides whether the registration may write. It does not change the router.
func PlanWan(reading Reading, request WanRequest) (WanPlan, error) {
	comment, ok := WanComment(request.Name)
	if !ok || !InterfaceSafe(request.InterfaceName) || request.Distance < 1 || request.Distance > 255 {
		return WanPlan{}, fmt.Errorf("invalid wan request")
	}
	active := ActiveRoute(reading.Routes)
	if active == nil || !InterfaceSafe(active.InterfaceName) {
		return WanPlan{}, &WanError{Reason: ReasonMissingActiveRoute}
	}
	ourDHCP := false
	foreignDistance := false
	for _, client := range reading.DHCPClients {
		if client.Comment == comment {
			if client.InterfaceName != request.InterfaceName || client.Distance != request.Distance {
				return WanPlan{}, &WanError{Reason: ReasonCommentTaken}
			}
			ourDHCP = true
			continue
		}
		if client.Distance == request.Distance {
			foreignDistance = true
		}
	}
	ourNAT := false
	for _, rule := range reading.NAT {
		if rule.Comment != comment {
			continue
		}
		if rule.InterfaceName != request.InterfaceName || rule.Chain != "srcnat" || rule.Action != "masquerade" {
			return WanPlan{}, &WanError{Reason: ReasonCommentTaken}
		}
		ourNAT = true
	}
	plan := WanPlan{
		Comment:        comment,
		InterfaceName:  request.InterfaceName,
		Distance:       request.Distance,
		AlreadyPresent: ourDHCP && ourNAT,
	}
	if plan.AlreadyPresent {
		return plan, nil
	}
	if !ourDHCP && !sparePort(reading, active.InterfaceName, request.InterfaceName) {
		return WanPlan{}, &WanError{Reason: ReasonNotSpare}
	}
	if !ourDHCP {
		for _, route := range reading.Routes {
			if route.Distance == request.Distance {
				return WanPlan{}, &WanError{Reason: ReasonDistanceTaken}
			}
		}
	}
	if foreignDistance {
		return WanPlan{}, &WanError{Reason: ReasonDistanceTaken}
	}
	return plan, nil
}

func sparePort(reading Reading, activeInterface string, iface string) bool {
	if iface == activeInterface {
		return false
	}
	for _, address := range reading.Addresses {
		if address.InterfaceName == iface {
			return false
		}
	}
	return true
}

// WanScripts are the one-line writes for a plan that still needs objects.
// RouterOS closes a brace only on the same line that opened it.
func WanScripts(plan WanPlan) ([]string, error) {
	if plan.AlreadyPresent {
		return []string{}, nil
	}
	if !strings.HasPrefix(plan.Comment, WanCommentPrefix) || strings.Contains(plan.Comment, PrimaryComment) {
		return nil, fmt.Errorf("invalid wan comment")
	}
	if !InterfaceSafe(plan.InterfaceName) || plan.Distance < 1 || plan.Distance > 255 {
		return nil, fmt.Errorf("invalid wan target")
	}
	distance := strconv.Itoa(plan.Distance)
	iface := plan.InterfaceName
	comment := plan.Comment
	scripts := []string{
		`:if ([:len [/interface bridge port find where interface="` + iface + `"]] > 0) do={ /interface bridge port remove [find where interface="` + iface + `"] }`,
		`:if ([:len [/ip dhcp-client find where comment="` + comment + `"]] = 0) do={ /ip dhcp-client add interface="` + iface + `" add-default-route=yes default-route-distance=` + distance + ` comment="` + comment + `" }`,
		`:if ([:len [/ip firewall nat find where comment="` + comment + `"]] = 0) do={ /ip firewall nat add chain=srcnat action=masquerade out-interface="` + iface + `" comment="` + comment + `" }`,
	}
	for _, script := range scripts {
		if strings.Contains(script, PrimaryComment) {
			return nil, fmt.Errorf("wan script names the primary route")
		}
		if strings.Contains(script, "{") && !strings.Contains(script, "}") {
			return nil, fmt.Errorf("wan script splits a block")
		}
	}
	return scripts, nil
}
