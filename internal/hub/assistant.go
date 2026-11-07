package hub

import (
	"context"
	"strings"
	"unicode/utf8"

	"github.com/lucaswilliameufrasio/borda/internal/assistant"
	"github.com/lucaswilliameufrasio/borda/internal/routeros"
)

const maxQuestionRunes = 2000

// Ask reads the current portrait and answers without executing a proposal.
func (store *Store) Ask(ctx context.Context, id string, question string) (assistant.Reply, error) {
	trimmed := strings.TrimSpace(question)
	if trimmed == "" {
		return assistant.Reply{}, &ValidationError{Fields: []FieldError{{
			Field:   "question",
			Message: "A pergunta está vazia.",
		}}}
	}
	if utf8.RuneCountInString(trimmed) > maxQuestionRunes {
		return assistant.Reply{}, &ValidationError{Fields: []FieldError{{
			Field:   "question",
			Message: "A pergunta passa de 2000 caracteres.",
		}}}
	}
	snapshot, err := store.Snapshot(ctx, id)
	if err != nil {
		return assistant.Reply{}, err
	}
	return assistant.Propose(assistant.PortraitOf(visibleReading(snapshot)), trimmed), nil
}

// ApplyProposal runs a closed-list action after confirmation.
// Any other action returns before the router is contacted.
func (store *Store) ApplyProposal(ctx context.Context, id string, action string, confirmed bool) (routeros.Proof, error) {
	if action != assistant.ActionProof {
		return routeros.Proof{}, &ValidationError{Fields: []FieldError{{
			Field:   "action",
			Message: "Essa proposta está fora da lista e não foi executada.",
		}}}
	}
	return store.Prove(ctx, id, confirmed)
}

func visibleReading(snapshot Snapshot) assistant.Reading {
	reading := assistant.Reading{Session: snapshot.Session}
	if snapshot.Identity != nil {
		reading.IdentityName = snapshot.Identity.Name
		reading.IdentityVersion = snapshot.Identity.Version
	}
	if snapshot.ActiveRoute != nil {
		reading.HasActiveRoute = true
		reading.ActiveGateway = snapshot.ActiveRoute.Gateway
		reading.ActiveInterface = snapshot.ActiveRoute.InterfaceName
		reading.ActiveDistance = snapshot.ActiveRoute.Distance
	}
	if snapshot.Check != nil {
		reading.HasCheck = true
		reading.CheckAddress = snapshot.Check.Address
		reading.CheckResponding = snapshot.Check.Responding
	}
	interfaces := make([]string, 0)
	distances := make([]int, 0)
	if snapshot.Routes != nil {
		reading.RouteCount = len(*snapshot.Routes)
		for _, route := range *snapshot.Routes {
			if route.InterfaceName != "" {
				interfaces = append(interfaces, route.InterfaceName)
			}
			distances = append(distances, route.Distance)
		}
	}
	comments := make([]string, 0)
	if snapshot.ForwardRules != nil {
		reading.ForwardCount = len(*snapshot.ForwardRules)
		for _, rule := range *snapshot.ForwardRules {
			if rule.Comment != "" {
				comments = append(comments, rule.Comment)
			}
			if rule.InInterface != "" {
				interfaces = append(interfaces, rule.InInterface)
			}
			if rule.OutInterface != "" {
				interfaces = append(interfaces, rule.OutInterface)
			}
		}
	}
	if snapshot.Leases != nil {
		reading.LeaseCount = len(*snapshot.Leases)
	}
	if snapshot.Devices != nil {
		count := 0
		for _, device := range *snapshot.Devices {
			if device.Presence == "" || device.Presence == PresenceNow {
				count = count + 1
			}
		}
		reading.DeviceCount = count
	}
	if snapshot.Reserve != nil {
		reading.ReserveRunning = snapshot.Reserve.Running
		reading.ReserveInterface = snapshot.Reserve.InterfaceName
		if snapshot.Reserve.InterfaceName != "" {
			interfaces = append(interfaces, snapshot.Reserve.InterfaceName)
		}
	}
	reading.Interfaces = uniqueStrings(interfaces)
	reading.Comments = comments
	reading.Distances = distances
	return reading
}

func uniqueStrings(values []string) []string {
	seen := map[string]struct{}{}
	unique := make([]string, 0, len(values))
	for _, value := range values {
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		unique = append(unique, value)
	}
	return unique
}
