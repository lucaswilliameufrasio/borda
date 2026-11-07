package assistant

import (
	"strconv"
	"strings"
)

const (
	KindAnswer   = "answer"
	KindProposal = "proposal"
	KindMissing  = "missing"
	KindRefused  = "refused"
	ActionProof  = "proof"
)

// Proposal is one action from the closed list. It is not an execution.
type Proposal struct {
	Action  string `json:"action"`
	Summary string `json:"summary"`
}

// Reply is what the person sees. A proposal still needs confirmation.
type Reply struct {
	Kind     string    `json:"kind"`
	Text     string    `json:"text"`
	Missing  []string  `json:"missing,omitempty"`
	Proposal *Proposal `json:"proposal,omitempty"`
}

// Propose answers from the portrait or returns a closed-list proposal.
// It does not talk to a router and it does not talk to a model.
func Propose(portrait Portrait, question string) Reply {
	cleaned := RedactQuestion(strings.TrimSpace(question))
	if asksForbidden(cleaned) {
		return Reply{
			Kind: KindRefused,
			Text: "Essa proposta está fora da lista e não foi executada.",
		}
	}
	if portrait.Session != "connected" {
		return Reply{
			Kind:    KindMissing,
			Text:    "A leitura não está disponível. Falta a sessão com o roteador.",
			Missing: []string{"session"},
		}
	}
	if !portrait.HasActiveRoute {
		return Reply{
			Kind:    KindMissing,
			Text:    "A leitura não mostra um caminho ativo.",
			Missing: []string{"active_route"},
		}
	}
	if asksProof(cleaned) {
		return Reply{
			Kind: KindProposal,
			Text: "Nada muda até a confirmação.",
			Proposal: &Proposal{
				Action:  ActionProof,
				Summary: "A prova desabilita só a rota comentada borda-primary, mede o destino pela USB e pela LAN, e reabilita a rota.",
			},
		}
	}
	return Reply{Kind: KindAnswer, Text: describe(portrait)}
}

func describe(portrait Portrait) string {
	exit := portrait.ActiveInterface
	if exit == "" {
		exit = "sem nome"
	}
	text := "O caminho ativo sai por " + exit + " para " + portrait.ActiveGateway + ", distância " + strconv.Itoa(portrait.ActiveDistance) + "."
	if portrait.HasCheck {
		if portrait.CheckResponding {
			text += " A checagem " + portrait.CheckAddress + " respondeu."
		} else {
			text += " A checagem " + portrait.CheckAddress + " não respondeu."
		}
	}
	text += " Há " + strconv.Itoa(portrait.LeaseCount) + " leases e " + strconv.Itoa(portrait.DeviceCount) + " aparelhos na leitura."
	if portrait.ReserveRunning && portrait.ReserveInterface != "" {
		text += " A reserva " + portrait.ReserveInterface + " está em pé."
	}
	return text
}

func asksForbidden(question string) bool {
	folded := Fold(question)
	if strings.Contains(folded, "add user") || strings.Contains(folded, "criar usuario") || strings.Contains(folded, "novo usuario") {
		return true
	}
	for _, token := range tokens(folded) {
		if token == "export" || token == "backup" || token == "rsc" || token == "senha" || token == "password" || token == "script" || token == "firewall" || token == "nat" || token == "usuario" {
			return true
		}
	}
	return false
}

func asksProof(question string) bool {
	for _, token := range tokens(Fold(question)) {
		if token == "prova" || token == "usb" || token == "failover" {
			return true
		}
	}
	return false
}

func tokens(value string) []string {
	return strings.FieldsFunc(value, func(symbol rune) bool {
		if symbol >= 'a' && symbol <= 'z' {
			return false
		}
		if symbol >= '0' && symbol <= '9' {
			return false
		}
		return true
	})
}
