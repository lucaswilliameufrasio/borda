package routeros

import (
	"fmt"
	"strings"
)

// ReadScript is the read-only RouterOS script for one snapshot.
func ReadScript(checkAddress string) (string, error) {
	var builder strings.Builder
	builder.WriteString(`:put ("identity` + "\t" + `" . [/system identity get name] . "` + "\t" + `" . [/system resource get version])` + "\n")
	builder.WriteString(`:foreach i in=[/ip route find where dst-address=0.0.0.0/0] do={` + "\n")
	builder.WriteString(`:put ("route` + "\t" + `" . [/ip route get $i gateway] . "` + "\t" + `" . [/ip route get $i active] . "` + "\t" + `" . [/ip route get $i distance] . "` + "\t" + `" . [/ip route get $i immediate-gw])` + "\n")
	builder.WriteString("}\n")
	builder.WriteString(`:foreach i in=[/ip firewall filter find where chain=forward] do={` + "\n")
	builder.WriteString(`:put ("forward` + "\t" + `" . [/ip firewall filter get $i action] . "` + "\t" + `" . [/ip firewall filter get $i comment] . "` + "\t" + `" . [/ip firewall filter get $i in-interface] . "` + "\t" + `" . [/ip firewall filter get $i out-interface])` + "\n")
	builder.WriteString("}\n")
	builder.WriteString(`:foreach i in=[/ip address find] do={` + "\n")
	builder.WriteString(`:put ("address` + "\t" + `" . [/ip address get $i interface] . "` + "\t" + `" . [/ip address get $i address])` + "\n")
	builder.WriteString("}\n")
	builder.WriteString(`:foreach i in=[/interface find where type="lte"] do={` + "\n")
	builder.WriteString(`:put ("lte` + "\t" + `" . [/interface get $i name] . "` + "\t" + `" . [/interface get $i running])` + "\n")
	builder.WriteString("}\n")
	builder.WriteString(`:foreach i in=[/ip dhcp-server lease find] do={` + "\n")
	builder.WriteString(`:put ("lease` + "\t" + `" . [/ip dhcp-server lease get $i address] . "` + "\t" + `" . [/ip dhcp-server lease get $i mac-address] . "` + "\t" + `" . [/ip dhcp-server lease get $i host-name])` + "\n")
	builder.WriteString("}\n")
	if checkAddress != "" {
		if !safeCheckAddress.MatchString(checkAddress) {
			return "", fmt.Errorf("invalid check address")
		}
		builder.WriteString(`:local result [/ping address="` + checkAddress + `" count=1 as-value]; :put ("check` + "\t" + `" . ($result->"received"))` + "\n")
	}
	script := builder.String()
	if strings.Contains(script, " add ") || strings.Contains(script, " set ") || strings.Contains(script, " remove ") {
		return "", fmt.Errorf("read script must not change the router")
	}
	return script, nil
}
