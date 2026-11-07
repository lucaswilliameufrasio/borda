package routeros

import (
	"fmt"
	"strings"
)

// ReadScript is the read-only RouterOS script for one snapshot.
// Each command stays on one line: RouterOS runs an SSH payload line by line,
// and a brace opened on one line is not closed by a later line.
// RouterOS 7 returns the ping summary key received only when proplist asks for it.
func ReadScript(checkAddress string) (string, error) {
	tab := "\t"
	lines := []string{
		`:put ("identity` + tab + `" . [/system identity get name] . "` + tab + `" . [/system resource get version])`,
		`:foreach i in=[/ip route find where dst-address=0.0.0.0/0] do={ :put ("route` + tab + `" . [/ip route get $i gateway] . "` + tab + `" . [/ip route get $i active] . "` + tab + `" . [/ip route get $i distance] . "` + tab + `" . [/ip route get $i immediate-gw] . "` + tab + `" . [/ip route get $i comment]) }`,
		`:foreach i in=[/ip firewall filter find where chain=forward] do={ :put ("forward` + tab + `" . [/ip firewall filter get $i action] . "` + tab + `" . [/ip firewall filter get $i comment] . "` + tab + `" . [/ip firewall filter get $i in-interface] . "` + tab + `" . [/ip firewall filter get $i out-interface]) }`,
		`:foreach i in=[/ip address find] do={ :put ("address` + tab + `" . [/ip address get $i interface] . "` + tab + `" . [/ip address get $i address]) }`,
		`:foreach i in=[/interface find where type="lte"] do={ :put ("lte` + tab + `" . [/interface get $i name] . "` + tab + `" . [/interface get $i running]) }`,
		`:foreach i in=[/ip dhcp-server lease find] do={ :put ("lease` + tab + `" . [/ip dhcp-server lease get $i address] . "` + tab + `" . [/ip dhcp-server lease get $i mac-address] . "` + tab + `" . [/ip dhcp-server lease get $i host-name]) }`,
	}
	if checkAddress != "" {
		if !safeCheckAddress.MatchString(checkAddress) {
			return "", fmt.Errorf("invalid check address")
		}
		lines = append(lines, `:local result [/ping address="`+checkAddress+`" count=1 proplist=received as-value]; :put ("check`+tab+`" . ($result->"received"))`)
	}
	var builder strings.Builder
	for _, line := range lines {
		builder.WriteString(line)
		builder.WriteString("\n")
	}
	script := builder.String()
	if strings.Contains(script, " add ") || strings.Contains(script, " set ") || strings.Contains(script, " remove ") {
		return "", fmt.Errorf("read script must not change the router")
	}
	return script, nil
}
