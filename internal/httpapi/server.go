package httpapi

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"net/http"
	"strings"

	"github.com/lucaswilliameufrasio/borda/internal/assistant"
	"github.com/lucaswilliameufrasio/borda/internal/hub"
	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

const maxBody = 1 << 16

// Server is the local read API.
type Server struct {
	store   *hub.Store
	logger  *log.Logger
	origins map[string]struct{}
	mux     *http.ServeMux
}

func New(store *hub.Store, logger *log.Logger, origins []string) *Server {
	if logger == nil {
		logger = log.New(io.Discard, "", 0)
	}
	allowed := map[string]struct{}{}
	for _, origin := range origins {
		allowed[origin] = struct{}{}
	}
	server := &Server{store: store, logger: logger, origins: allowed, mux: http.NewServeMux()}
	server.mux.HandleFunc("GET /healthz", server.health)
	server.mux.HandleFunc("GET /v1/routers", server.listRouters)
	server.mux.HandleFunc("POST /v1/routers", server.createRouter)
	server.mux.HandleFunc("GET /v1/routers/{id}", server.getRouter)
	server.mux.HandleFunc("GET /v1/routers/{id}/snapshot", server.snapshot)
	server.mux.HandleFunc("GET /v1/routers/{id}/events", server.events)
	server.mux.HandleFunc("POST /v1/routers/{id}/session", server.unlock)
	server.mux.HandleFunc("POST /v1/routers/{id}/proof", server.proof)
	server.mux.HandleFunc("POST /v1/routers/{id}/assistant", server.assistant)
	server.mux.HandleFunc("POST /v1/routers/{id}/assistant/actions", server.assistantAction)
	return server
}

func (server *Server) Handler() http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		recorder := &statusRecorder{ResponseWriter: response, status: http.StatusOK}
		server.cors(server.mux).ServeHTTP(recorder, request)
		server.logger.Printf("%s %s %d", request.Method, request.URL.Path, recorder.status)
	})
}

func (server *Server) cors(next http.Handler) http.Handler {
	return http.HandlerFunc(func(response http.ResponseWriter, request *http.Request) {
		origin := request.Header.Get("Origin")
		if origin != "" && server.originAllowed(origin) {
			response.Header().Set("Access-Control-Allow-Origin", origin)
			response.Header().Set("Vary", "Origin")
			response.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
			response.Header().Set("Access-Control-Allow-Headers", "Content-Type, Accept")
			response.Header().Set("Access-Control-Max-Age", "600")
		}
		if request.Method == http.MethodOptions {
			if origin != "" && !server.originAllowed(origin) {
				writeError(response, http.StatusForbidden, "PERMISSION_DENIED", "Origem não permitida.", nil)
				return
			}
			response.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(response, request)
	})
}

func (server *Server) originAllowed(origin string) bool {
	_, ok := server.origins[origin]
	return ok
}

func (server *Server) health(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, map[string]string{"status": "ok"})
}

func (server *Server) listRouters(response http.ResponseWriter, request *http.Request) {
	writeJSON(response, http.StatusOK, server.store.List())
}

func (server *Server) getRouter(response http.ResponseWriter, request *http.Request) {
	view, err := server.store.Get(request.PathValue("id"))
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, view)
}

func (server *Server) createRouter(response http.ResponseWriter, request *http.Request) {
	var body registerBody
	if err := decodeJSON(response, request, &body); err != nil {
		writeDecodeError(response, err)
		return
	}
	input := hub.RegisterInput{
		Name:            body.Name,
		Host:            body.Host,
		Username:        body.Username,
		CheckAddress:    body.CheckAddress,
		VaultKind:       body.VaultKind,
		RouterPassword:  body.RouterPassword,
		VaultPassphrase: body.VaultPassphrase,
		APIURL:          body.APIURL,
		ProjectID:       body.ProjectID,
		Environment:     body.Environment,
		SecretKey:       body.SecretKey,
		SecretName:      body.SecretName,
		AccessToken:     body.AccessToken,
	}
	if body.SSHPort != nil {
		input.SSHPort = *body.SSHPort
		input.SSHPortSet = true
	}
	view, err := server.store.Register(request.Context(), input)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusCreated, view)
}

func (server *Server) snapshot(response http.ResponseWriter, request *http.Request) {
	snapshot, err := server.store.Snapshot(request.Context(), request.PathValue("id"))
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, snapshot)
}

func (server *Server) events(response http.ResponseWriter, request *http.Request) {
	events, err := server.store.Events(request.PathValue("id"))
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, events)
}

func (server *Server) proof(response http.ResponseWriter, request *http.Request) {
	var body proofBody
	if err := decodeJSON(response, request, &body); err != nil {
		writeDecodeError(response, err)
		return
	}
	proof, err := server.store.Prove(request.Context(), request.PathValue("id"), body.Confirmed)
	server.writeProof(response, proof, err)
}

func (server *Server) assistant(response http.ResponseWriter, request *http.Request) {
	var body assistantBody
	if err := decodeJSON(response, request, &body); err != nil {
		writeDecodeError(response, err)
		return
	}
	reply, err := server.store.Ask(request.Context(), request.PathValue("id"), body.Question)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, assistantResponse(reply))
}

func (server *Server) assistantAction(response http.ResponseWriter, request *http.Request) {
	var body assistantActionBody
	if err := decodeJSON(response, request, &body); err != nil {
		writeDecodeError(response, err)
		return
	}
	proof, err := server.store.ApplyProposal(request.Context(), request.PathValue("id"), body.Action, body.Confirmed)
	server.writeProof(response, proof, err)
}

func (server *Server) writeProof(response http.ResponseWriter, proof routeros.Proof, err error) {
	if err != nil {
		var notReady *routeros.ProofError
		if errors.As(err, &notReady) {
			writeJSON(response, http.StatusPreconditionFailed, map[string]any{
				"message":    "A prova não começou e a rota primária não foi alterada.",
				"error_code": "PROOF_NOT_READY",
				"extra":      map[string]string{"reason": notReady.Reason},
			})
			return
		}
		if errors.Is(err, routeros.ErrProofNotRestored) {
			writeJSON(response, http.StatusInternalServerError, map[string]any{
				"message":    "A rota primária não voltou a ficar ativa.",
				"error_code": "PROOF_NOT_RESTORED",
			})
			return
		}
		writeStoreError(response, err)
		return
	}
	writeJSON(response, http.StatusOK, map[string]any{
		"outcome":        proof.Outcome,
		"primary_active": proof.PrimaryActive,
		"usb_received":   proof.USBReceived,
		"lan_received":   proof.LANReceived,
	})
}

func assistantResponse(reply assistant.Reply) assistantReplyBody {
	body := assistantReplyBody{Kind: reply.Kind, Text: reply.Text}
	if reply.Kind == assistant.KindMissing {
		body.Missing = reply.Missing
		if body.Missing == nil {
			body.Missing = []string{}
		}
	}
	if reply.Proposal != nil {
		body.Proposal = &assistantProposalBody{Action: reply.Proposal.Action, Summary: reply.Proposal.Summary}
	}
	return body
}

func (server *Server) unlock(response http.ResponseWriter, request *http.Request) {
	var body unlockBody
	if err := decodeJSON(response, request, &body); err != nil {
		writeDecodeError(response, err)
		return
	}
	err := server.store.Unlock(request.Context(), request.PathValue("id"), body.VaultPassphrase, body.AccessToken)
	if err != nil {
		writeStoreError(response, err)
		return
	}
	response.WriteHeader(http.StatusNoContent)
}

type registerBody struct {
	Name            string `json:"name"`
	Host            string `json:"host"`
	SSHPort         *int   `json:"ssh_port"`
	Username        string `json:"username"`
	CheckAddress    string `json:"check_address"`
	VaultKind       string `json:"vault_kind"`
	RouterPassword  string `json:"router_password"`
	VaultPassphrase string `json:"vault_passphrase"`
	APIURL          string `json:"api_url"`
	ProjectID       string `json:"project_id"`
	Environment     string `json:"environment"`
	SecretKey       string `json:"secret_key"`
	SecretName      string `json:"secret_name"`
	AccessToken     string `json:"access_token"`
}

type unlockBody struct {
	VaultPassphrase string `json:"vault_passphrase"`
	AccessToken     string `json:"access_token"`
}

type proofBody struct {
	Confirmed bool `json:"confirmed"`
}

type assistantBody struct {
	Question string `json:"question"`
}

type assistantActionBody struct {
	Action    string `json:"action"`
	Confirmed bool   `json:"confirmed"`
}

type assistantReplyBody struct {
	Kind     string                 `json:"kind"`
	Text     string                 `json:"text"`
	Missing  []string               `json:"missing,omitempty"`
	Proposal *assistantProposalBody `json:"proposal,omitempty"`
}

type assistantProposalBody struct {
	Action  string `json:"action"`
	Summary string `json:"summary"`
}

type errorBody struct {
	Message   string      `json:"message"`
	ErrorCode string      `json:"error_code"`
	Extra     *errorExtra `json:"extra,omitempty"`
}

type errorExtra struct {
	ValidationErrors []fieldError `json:"validation_errors"`
}

type fieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func writeStoreError(response http.ResponseWriter, err error) {
	var validation *hub.ValidationError
	if errors.As(err, &validation) {
		fields := make([]fieldError, 0, len(validation.Fields))
		for _, field := range validation.Fields {
			fields = append(fields, fieldError{Field: field.Field, Message: field.Message})
		}
		writeError(response, http.StatusUnprocessableEntity, "INVALID_PARAMS", "Os dados do roteador não passaram na validação.", fields)
		return
	}
	if errors.Is(err, hub.ErrNotFound) {
		writeError(response, http.StatusNotFound, "ROUTER_NOT_FOUND", "Roteador não encontrado.", nil)
		return
	}
	if errors.Is(err, secret.ErrLocked) {
		writeError(response, http.StatusUnprocessableEntity, "INVALID_PARAMS", "Os dados do roteador não passaram na validação.", []fieldError{{
			Field:   "vault",
			Message: "O cofre não abriu.",
		}})
		return
	}
	if errors.Is(err, secret.ErrNotFound) {
		writeError(response, http.StatusUnprocessableEntity, "INVALID_PARAMS", "Os dados do roteador não passaram na validação.", []fieldError{{
			Field:   "vault",
			Message: "O cofre não tem esse segredo.",
		}})
		return
	}
	if errors.Is(err, secret.ErrUnavailable) {
		writeError(response, http.StatusBadGateway, "VAULT_UNAVAILABLE", "O cofre não respondeu.", nil)
		return
	}
	writeError(response, http.StatusInternalServerError, "UNEXPECTED_ERROR", "Não foi possível concluir o pedido.", nil)
}

func writeDecodeError(response http.ResponseWriter, err error) {
	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		writeError(response, http.StatusRequestEntityTooLarge, "PAYLOAD_TOO_LARGE", "O pedido é grande demais.", nil)
		return
	}
	if strings.Contains(err.Error(), "unknown field") {
		writeError(response, http.StatusUnprocessableEntity, "INVALID_PARAMS", "Os dados do roteador não passaram na validação.", []fieldError{{
			Field:   "body",
			Message: "O pedido tem um campo desconhecido.",
		}})
		return
	}
	writeError(response, http.StatusBadRequest, "MALFORMED_REQUEST", "O corpo do pedido não é JSON válido.", nil)
}

func writeError(response http.ResponseWriter, status int, code string, message string, fields []fieldError) {
	body := errorBody{Message: message, ErrorCode: code}
	if fields != nil {
		if len(fields) == 0 {
			fields = []fieldError{}
		}
		body.Extra = &errorExtra{ValidationErrors: fields}
	}
	writeJSON(response, status, body)
}

func writeJSON(response http.ResponseWriter, status int, payload any) {
	response.Header().Set("Content-Type", "application/json; charset=utf-8")
	response.WriteHeader(status)
	encoder := json.NewEncoder(response)
	encoder.SetEscapeHTML(false)
	if err := encoder.Encode(payload); err != nil {
		return
	}
}

func decodeJSON(response http.ResponseWriter, request *http.Request, destination any) error {
	request.Body = http.MaxBytesReader(response, request.Body, maxBody)
	decoder := json.NewDecoder(request.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(destination); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return errors.New("trailing json")
	}
	return nil
}

type statusRecorder struct {
	http.ResponseWriter
	status int
	wrote  bool
}

func (recorder *statusRecorder) WriteHeader(status int) {
	recorder.status = status
	recorder.wrote = true
	recorder.ResponseWriter.WriteHeader(status)
}

func (recorder *statusRecorder) Write(body []byte) (int, error) {
	if !recorder.wrote {
		recorder.status = http.StatusOK
		recorder.wrote = true
	}
	return recorder.ResponseWriter.Write(body)
}
