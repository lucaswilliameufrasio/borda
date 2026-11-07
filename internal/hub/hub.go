package hub

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/lucaswilliameufrasio/borda/internal/routeros"
	"github.com/lucaswilliameufrasio/borda/internal/secret"
)

var ErrNotFound = errors.New("router not found")

var endpointPattern = regexp.MustCompile(`^[A-Za-z0-9.:-]{1,253}$`)
var idPattern = regexp.MustCompile(`^[0-9a-f]{32}$`)
var tokenPattern = regexp.MustCompile(`^[A-Za-z0-9._:-]{1,128}$`)

// FieldError is one validation failure.
type FieldError struct {
	Field   string
	Message string
}

// ValidationError is a set of field failures.
type ValidationError struct {
	Fields []FieldError
}

func (err *ValidationError) Error() string {
	return "invalid params"
}

// RegisterInput is the setup form. Secret fields are not persisted.
type RegisterInput struct {
	Name            string
	Host            string
	SSHPort         int
	SSHPortSet      bool
	Username        string
	CheckAddress    string
	VaultKind       string
	RouterPassword  string
	VaultPassphrase string
	APIURL          string
	ProjectID       string
	Environment     string
	SecretKey       string
	SecretName      string
	AccessToken     string
}

// RouterView is the public router record.
type RouterView struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	Host         string `json:"host"`
	SSHPort      int    `json:"ssh_port"`
	CheckAddress string `json:"check_address,omitempty"`
	VaultKind    string `json:"vault_kind"`
}

type storedRouter struct {
	ID           string        `json:"id"`
	Name         string        `json:"name"`
	Host         string        `json:"host"`
	SSHPort      int           `json:"ssh_port"`
	Username     string        `json:"username"`
	CheckAddress string        `json:"check_address,omitempty"`
	Vault        secret.Record `json:"vault"`
}

type fileState struct {
	Routers []storedRouter `json:"routers"`
}

// Store is the local router registry. Passwords stay in the keeper.
type Store struct {
	dir          string
	path         string
	historyPath  string
	linksPath    string
	presencePath string
	keeper       *secret.Keeper
	reader       routeros.Reader
	now          func() time.Time

	mu           sync.Mutex
	routers      []storedRouter
	observations map[string]pathObservation
	events       []storedEvent
	links        []storedLink
	presence     []storedPresence
}

func Open(dir string, keeper *secret.Keeper, reader routeros.Reader) (*Store, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	store := &Store{
		dir:          dir,
		path:         filepath.Join(dir, "routers.json"),
		historyPath:  filepath.Join(dir, "events.json"),
		linksPath:    filepath.Join(dir, "links.json"),
		presencePath: filepath.Join(dir, "presence.json"),
		keeper:       keeper,
		reader:       reader,
		routers:      []storedRouter{},
		observations: map[string]pathObservation{},
		events:       []storedEvent{},
		links:        []storedLink{},
		presence:     []storedPresence{},
	}
	if err := store.load(); err != nil {
		return nil, err
	}
	if err := store.loadHistory(); err != nil {
		return nil, err
	}
	if err := store.loadLinks(); err != nil {
		return nil, err
	}
	if err := store.loadPresence(); err != nil {
		return nil, err
	}
	return store, nil
}

func (store *Store) List() []RouterView {
	store.mu.Lock()
	defer store.mu.Unlock()
	views := make([]RouterView, 0, len(store.routers))
	for _, router := range store.routers {
		views = append(views, publicView(router))
	}
	return views
}

func (store *Store) Get(id string) (RouterView, error) {
	router, err := store.find(id)
	if err != nil {
		return RouterView{}, err
	}
	return publicView(router), nil
}

func (store *Store) Register(ctx context.Context, input RegisterInput) (RouterView, error) {
	normalized, fields := normalize(input)
	if len(fields) > 0 {
		return RouterView{}, &ValidationError{Fields: fields}
	}
	record := vaultRecord(normalized)
	id, err := newID()
	if err != nil {
		return RouterView{}, err
	}
	material := secret.Material{
		Passphrase:  normalized.VaultPassphrase,
		AccessToken: normalized.AccessToken,
	}
	if err := store.keeper.Put(ctx, id, record, material, normalized.RouterPassword); err != nil {
		return RouterView{}, err
	}
	router := storedRouter{
		ID:           id,
		Name:         normalized.Name,
		Host:         normalized.Host,
		SSHPort:      normalized.SSHPort,
		Username:     normalized.Username,
		CheckAddress: normalized.CheckAddress,
		Vault:        record,
	}
	store.mu.Lock()
	store.routers = append(store.routers, router)
	err = store.saveLocked()
	if err != nil {
		store.routers = store.routers[:len(store.routers)-1]
	}
	store.mu.Unlock()
	if err != nil {
		store.keeper.Forget(id)
		return RouterView{}, err
	}
	return publicView(router), nil
}

func (store *Store) Unlock(ctx context.Context, id string, passphrase string, token string) error {
	router, err := store.find(id)
	if err != nil {
		return err
	}
	fields := unlockFields(router.Vault.Kind, passphrase, token)
	if len(fields) > 0 {
		return &ValidationError{Fields: fields}
	}
	store.keeper.Remember(id, secret.Material{Passphrase: passphrase, AccessToken: token})
	if _, err := store.keeper.Get(ctx, id, router.Vault); err != nil {
		store.keeper.Forget(id)
		return err
	}
	return nil
}

func (store *Store) Snapshot(ctx context.Context, id string) (Snapshot, error) {
	router, err := store.find(id)
	if err != nil {
		return Snapshot{}, err
	}
	snapshot := Snapshot{
		RouterID: router.ID,
		Name:     router.Name,
		Session:  "connected",
	}
	password, err := store.keeper.Get(ctx, router.ID, router.Vault)
	if err != nil {
		snapshot.Session = "locked"
		return snapshot, nil
	}
	reading, err := store.reader.Read(ctx, routeros.Target{
		Host:         router.Host,
		Port:         router.SSHPort,
		Username:     router.Username,
		Password:     password,
		CheckAddress: router.CheckAddress,
	})
	if err != nil {
		snapshot.Session = sessionFromReader(err)
		return snapshot, nil
	}
	filled := fillConnected(snapshot, router, reading)
	if err := store.notePath(router.ID, filled); err != nil {
		return Snapshot{}, err
	}
	if filled.Devices == nil {
		empty := []DeviceJSON{}
		filled.Devices = &empty
	}
	devices, err := store.notePresence(router.ID, *filled.Devices)
	if err != nil {
		return Snapshot{}, err
	}
	filled.Devices = &devices
	return filled, nil
}

// Prove runs the USB proof after an explicit confirmation.
// Without confirmation it returns a validation error and does not call the router.
func (store *Store) Prove(ctx context.Context, id string, confirmed bool) (routeros.Proof, error) {
	if !confirmed {
		return routeros.Proof{}, &ValidationError{Fields: []FieldError{{
			Field:   "confirmed",
			Message: "A prova só começa com confirmação.",
		}}}
	}
	router, err := store.find(id)
	if err != nil {
		return routeros.Proof{}, err
	}
	password, err := store.keeper.Get(ctx, router.ID, router.Vault)
	if err != nil {
		return routeros.Proof{}, err
	}
	started := store.clock()
	proof, err := store.reader.Prove(ctx, routeros.Target{
		Host:         router.Host,
		Port:         router.SSHPort,
		Username:     router.Username,
		Password:     password,
		CheckAddress: router.CheckAddress,
	})
	if err != nil {
		var notReady *routeros.ProofError
		if errors.As(err, &notReady) {
			return proof, err
		}
		if !errors.Is(err, routeros.ErrProofNotRestored) {
			return proof, err
		}
		if noteErr := store.noteProof(router.ID, started, false); noteErr != nil {
			return proof, noteErr
		}
		return proof, err
	}
	if noteErr := store.noteProof(router.ID, started, true); noteErr != nil {
		return proof, noteErr
	}
	return proof, nil
}

// RegisterWAN creates one WAN after an explicit confirmation.
// Without confirmation it returns a validation error and does not call the router.
func (store *Store) RegisterWAN(ctx context.Context, id string, name string, iface string, distance int, confirmed bool) (routeros.WanResult, error) {
	if !confirmed {
		return routeros.WanResult{}, &ValidationError{Fields: []FieldError{{
			Field:   "confirmed",
			Message: "A WAN só é cadastrada com confirmação.",
		}}}
	}
	if _, ok := routeros.WanComment(name); !ok {
		return routeros.WanResult{}, &ValidationError{Fields: []FieldError{{
			Field:   "name",
			Message: "O nome da WAN usa letras, números e hífen, e começa com letra.",
		}}}
	}
	if !routeros.InterfaceSafe(iface) {
		return routeros.WanResult{}, &ValidationError{Fields: []FieldError{{
			Field:   "interface_name",
			Message: "A porta só pode ter letras, números, ponto, dois-pontos e hífen.",
		}}}
	}
	if distance < 1 || distance > 255 {
		return routeros.WanResult{}, &ValidationError{Fields: []FieldError{{
			Field:   "distance",
			Message: "A distância fica entre 1 e 255.",
		}}}
	}
	router, err := store.find(id)
	if err != nil {
		return routeros.WanResult{}, err
	}
	password, err := store.keeper.Get(ctx, router.ID, router.Vault)
	if err != nil {
		return routeros.WanResult{}, err
	}
	return store.reader.RegisterWAN(ctx, routeros.Target{
		Host:         router.Host,
		Port:         router.SSHPort,
		Username:     router.Username,
		Password:     password,
		CheckAddress: router.CheckAddress,
	}, routeros.WanRequest{
		Name:          name,
		InterfaceName: iface,
		Distance:      distance,
	})
}

func (store *Store) find(id string) (storedRouter, error) {
	store.mu.Lock()
	defer store.mu.Unlock()
	for _, router := range store.routers {
		if router.ID == id {
			return router, nil
		}
	}
	return storedRouter{}, ErrNotFound
}

func (store *Store) load() error {
	body, err := os.ReadFile(store.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	var state fileState
	if err := json.Unmarshal(body, &state); err != nil {
		return err
	}
	if state.Routers == nil {
		state.Routers = []storedRouter{}
	}
	for _, router := range state.Routers {
		if !idPattern.MatchString(router.ID) {
			return errors.New("invalid router id in registry")
		}
	}
	store.routers = state.Routers
	return nil
}

func (store *Store) saveLocked() error {
	state := fileState{Routers: store.routers}
	if state.Routers == nil {
		state.Routers = []storedRouter{}
	}
	body, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return err
	}
	body = append(body, '\n')
	temporary := store.path + ".tmp"
	if err := os.WriteFile(temporary, body, 0o600); err != nil {
		return err
	}
	return os.Rename(temporary, store.path)
}

func publicView(router storedRouter) RouterView {
	return RouterView{
		ID:           router.ID,
		Name:         router.Name,
		Host:         router.Host,
		SSHPort:      router.SSHPort,
		CheckAddress: router.CheckAddress,
		VaultKind:    router.Vault.Kind,
	}
}

func vaultRecord(input RegisterInput) secret.Record {
	record := secret.Record{Kind: input.VaultKind}
	if input.VaultKind == secret.KindCiphera || input.VaultKind == secret.KindInfisical {
		record.APIURL = input.APIURL
		record.ProjectID = input.ProjectID
		record.Environment = input.Environment
	}
	if input.VaultKind == secret.KindCiphera {
		record.SecretKey = input.SecretKey
	}
	if input.VaultKind == secret.KindInfisical {
		record.SecretName = input.SecretName
	}
	return record
}

func normalize(input RegisterInput) (RegisterInput, []FieldError) {
	input.Name = strings.TrimSpace(input.Name)
	input.Host = strings.TrimSpace(input.Host)
	input.Username = strings.TrimSpace(input.Username)
	input.CheckAddress = strings.TrimSpace(input.CheckAddress)
	input.APIURL = strings.TrimSpace(input.APIURL)
	input.ProjectID = strings.TrimSpace(input.ProjectID)
	input.Environment = strings.TrimSpace(input.Environment)
	input.SecretKey = strings.TrimSpace(input.SecretKey)
	input.SecretName = strings.TrimSpace(input.SecretName)
	input.AccessToken = strings.TrimSpace(input.AccessToken)
	if !input.SSHPortSet {
		input.SSHPort = 22
	}
	fields := make([]FieldError, 0)
	if input.Name == "" || len(input.Name) > 80 || strings.ContainsAny(input.Name, "\r\n\t") {
		fields = append(fields, FieldError{Field: "name", Message: "Informe um nome com até 80 caracteres."})
	}
	if !endpointPattern.MatchString(input.Host) || strings.HasPrefix(input.Host, "-") {
		fields = append(fields, FieldError{Field: "host", Message: "Informe o endereço do roteador."})
	}
	if input.SSHPort < 1 || input.SSHPort > 65535 {
		fields = append(fields, FieldError{Field: "ssh_port", Message: "A porta SSH precisa estar entre 1 e 65535."})
	}
	if !endpointPattern.MatchString(input.Username) || strings.HasPrefix(input.Username, "-") {
		fields = append(fields, FieldError{Field: "username", Message: "Informe o usuário do roteador."})
	}
	if input.CheckAddress != "" && !routeros.CheckAddressSafe(input.CheckAddress) {
		fields = append(fields, FieldError{Field: "check_address", Message: "O endereço de checagem só pode ter letras, números, ponto, dois-pontos e hífen."})
	}
	switch input.VaultKind {
	case secret.KindFile:
		fields = append(fields, requireSecret(input.RouterPassword, "router_password", "Informe a senha do roteador.")...)
		fields = append(fields, requireSecret(input.VaultPassphrase, "vault_passphrase", "Informe a senha do cofre.")...)
	case secret.KindKeychain:
		fields = append(fields, requireSecret(input.RouterPassword, "router_password", "Informe a senha do roteador.")...)
	case secret.KindCiphera:
		fields = append(fields, remoteVaultFields(input, true)...)
	case secret.KindInfisical:
		fields = append(fields, remoteVaultFields(input, false)...)
	default:
		fields = append(fields, FieldError{Field: "vault_kind", Message: "Escolha um cofre: arquivo, chaveiro, Ciphera ou Infisical."})
	}
	return input, fields
}

func remoteVaultFields(input RegisterInput, ciphera bool) []FieldError {
	fields := make([]FieldError, 0)
	parsed, err := url.Parse(input.APIURL)
	if err != nil || parsed.Host == "" || parsed.User != nil || (parsed.Scheme != "https" && parsed.Scheme != "http") {
		fields = append(fields, FieldError{Field: "api_url", Message: "Informe a URL da API do cofre, sem usuário nem senha."})
	}
	if !tokenPattern.MatchString(input.ProjectID) {
		fields = append(fields, FieldError{Field: "project_id", Message: "Informe o projeto."})
	}
	if !tokenPattern.MatchString(input.Environment) {
		fields = append(fields, FieldError{Field: "environment", Message: "Informe o ambiente."})
	}
	if ciphera {
		if !tokenPattern.MatchString(input.SecretKey) {
			fields = append(fields, FieldError{Field: "secret_key", Message: "Informe a chave do segredo."})
		}
	}
	if !ciphera {
		if !tokenPattern.MatchString(input.SecretName) {
			fields = append(fields, FieldError{Field: "secret_name", Message: "Informe o nome do segredo."})
		}
	}
	fields = append(fields, requireSecret(input.AccessToken, "access_token", "Informe o token de acesso.")...)
	return fields
}

func requireSecret(value string, field string, message string) []FieldError {
	if value == "" || len(value) > 4096 || strings.ContainsAny(value, "\r\n") {
		return []FieldError{{Field: field, Message: message}}
	}
	return []FieldError{}
}

func unlockFields(kind string, passphrase string, token string) []FieldError {
	if kind == secret.KindFile {
		return requireSecret(passphrase, "vault_passphrase", "Informe a senha do cofre.")
	}
	if kind == secret.KindCiphera || kind == secret.KindInfisical {
		return requireSecret(token, "access_token", "Informe o token de acesso.")
	}
	return []FieldError{}
}

func sessionFromReader(err error) string {
	if errors.Is(err, routeros.ErrAuthenticationRejected) {
		return "authentication_rejected"
	}
	return "unreachable"
}

func newID() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}
