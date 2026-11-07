package assistant

import (
	"encoding/json"
	"regexp"
	"strings"
)

// Portrait is the only router view a proposer may see.
type Portrait struct {
	Session          string   `json:"session"`
	IdentityName     string   `json:"identity_name,omitempty"`
	IdentityVersion  string   `json:"identity_version,omitempty"`
	ActiveGateway    string   `json:"active_gateway,omitempty"`
	ActiveInterface  string   `json:"active_interface,omitempty"`
	ActiveDistance   int      `json:"active_distance,omitempty"`
	HasActiveRoute   bool     `json:"has_active_route"`
	CheckAddress     string   `json:"check_address,omitempty"`
	CheckResponding  bool     `json:"check_responding"`
	HasCheck         bool     `json:"has_check"`
	RouteCount       int      `json:"route_count"`
	ForwardCount     int      `json:"forward_count"`
	LeaseCount       int      `json:"lease_count"`
	DeviceCount      int      `json:"device_count"`
	Interfaces       []string `json:"interfaces"`
	Comments         []string `json:"comments"`
	Distances        []int    `json:"distances"`
	ReserveInterface string   `json:"reserve_interface,omitempty"`
	ReserveRunning   bool     `json:"reserve_running"`
}

// Reading is the allowlisted operational view used to build a portrait.
type Reading struct {
	Session          string
	IdentityName     string
	IdentityVersion  string
	ActiveGateway    string
	ActiveInterface  string
	ActiveDistance   int
	HasActiveRoute   bool
	CheckAddress     string
	CheckResponding  bool
	HasCheck         bool
	RouteCount       int
	ForwardCount     int
	LeaseCount       int
	DeviceCount      int
	Interfaces       []string
	Comments         []string
	Distances        []int
	ReserveInterface string
	ReserveRunning   bool
}

type promptDocument struct {
	Portrait Portrait `json:"portrait"`
	Question string   `json:"question"`
}

type looseDocument struct {
	Session          string   `json:"session"`
	IdentityName     string   `json:"identity_name"`
	IdentityVersion  string   `json:"identity_version"`
	ActiveGateway    string   `json:"active_gateway"`
	ActiveInterface  string   `json:"active_interface"`
	ActiveDistance   int      `json:"active_distance"`
	HasActiveRoute   bool     `json:"has_active_route"`
	CheckAddress     string   `json:"check_address"`
	CheckResponding  bool     `json:"check_responding"`
	HasCheck         bool     `json:"has_check"`
	RouteCount       int      `json:"route_count"`
	ForwardCount     int      `json:"forward_count"`
	LeaseCount       int      `json:"lease_count"`
	DeviceCount      int      `json:"device_count"`
	Interfaces       []string `json:"interfaces"`
	Comments         []string `json:"comments"`
	Distances        []int    `json:"distances"`
	ReserveInterface string   `json:"reserve_interface"`
	ReserveRunning   bool     `json:"reserve_running"`

	Password      string `json:"password"`
	Username      string `json:"username"`
	Export        string `json:"export"`
	Backup        string `json:"backup"`
	Script        string `json:"script"`
	CredentialURL string `json:"credential_url"`
	HostName      string `json:"host_name"`
	MAC           string `json:"mac"`
	UserContent   string `json:"user_content"`
}

var credentialURL = regexp.MustCompile(`(?i)[a-z][a-z0-9+.-]*://[^/\s:@]+:[^/\s@]+@`)

// PortraitOf copies the allowlisted reading into the prompt view.
func PortraitOf(reading Reading) Portrait {
	return normalize(Portrait{
		Session:          reading.Session,
		IdentityName:     reading.IdentityName,
		IdentityVersion:  reading.IdentityVersion,
		ActiveGateway:    reading.ActiveGateway,
		ActiveInterface:  reading.ActiveInterface,
		ActiveDistance:   reading.ActiveDistance,
		HasActiveRoute:   reading.HasActiveRoute,
		CheckAddress:     reading.CheckAddress,
		CheckResponding:  reading.CheckResponding,
		HasCheck:         reading.HasCheck,
		RouteCount:       reading.RouteCount,
		ForwardCount:     reading.ForwardCount,
		LeaseCount:       reading.LeaseCount,
		DeviceCount:      reading.DeviceCount,
		Interfaces:       reading.Interfaces,
		Comments:         reading.Comments,
		Distances:        reading.Distances,
		ReserveInterface: reading.ReserveInterface,
		ReserveRunning:   reading.ReserveRunning,
	})
}

// PortraitFromLoose keeps only allowlisted fields from a candidate document.
func PortraitFromLoose(raw []byte) (Portrait, error) {
	var loose looseDocument
	if err := json.Unmarshal(raw, &loose); err != nil {
		return Portrait{}, err
	}
	return PortraitOf(Reading{
		Session:          loose.Session,
		IdentityName:     loose.IdentityName,
		IdentityVersion:  loose.IdentityVersion,
		ActiveGateway:    loose.ActiveGateway,
		ActiveInterface:  loose.ActiveInterface,
		ActiveDistance:   loose.ActiveDistance,
		HasActiveRoute:   loose.HasActiveRoute,
		CheckAddress:     loose.CheckAddress,
		CheckResponding:  loose.CheckResponding,
		HasCheck:         loose.HasCheck,
		RouteCount:       loose.RouteCount,
		ForwardCount:     loose.ForwardCount,
		LeaseCount:       loose.LeaseCount,
		DeviceCount:      loose.DeviceCount,
		Interfaces:       loose.Interfaces,
		Comments:         loose.Comments,
		Distances:        loose.Distances,
		ReserveInterface: loose.ReserveInterface,
		ReserveRunning:   loose.ReserveRunning,
	}), nil
}

// Prompt is the document a proposer receives. Credential URLs in the question are removed.
func Prompt(portrait Portrait, question string) ([]byte, error) {
	return json.Marshal(promptDocument{
		Portrait: normalize(portrait),
		Question: RedactQuestion(question),
	})
}

// RedactQuestion removes URLs that carry a user and a password.
func RedactQuestion(question string) string {
	return credentialURL.ReplaceAllString(question, "")
}

func normalize(portrait Portrait) Portrait {
	if portrait.Interfaces == nil {
		portrait.Interfaces = []string{}
	}
	if portrait.Comments == nil {
		portrait.Comments = []string{}
	}
	if portrait.Distances == nil {
		portrait.Distances = []int{}
	}
	return portrait
}

// Fold prepares a question for keyword checks.
func Fold(value string) string {
	replacer := strings.NewReplacer(
		"á", "a", "à", "a", "ã", "a", "â", "a",
		"é", "e", "ê", "e",
		"í", "i",
		"ó", "o", "ô", "o", "õ", "o",
		"ú", "u",
		"ç", "c",
	)
	return replacer.Replace(strings.ToLower(value))
}
