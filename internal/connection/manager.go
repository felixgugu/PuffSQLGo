package connection

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"puffsql/internal/model"
	"puffsql/internal/mssql"
	"puffsql/internal/storage"
)

type Manager struct {
	store       *storage.Store
	credentials CredentialStore
	mu          sync.RWMutex
	servers     map[string]*mssql.Server
	passwords   map[string]string
	spids       map[string]int
	queries     map[string]*activeQuery
}

type activeQuery struct {
	connectionID string
	cancel       context.CancelFunc
	spid         int
}

func NewManager(store *storage.Store) *Manager {
	return &Manager{
		store:       store,
		credentials: OSKeyring{},
		servers:     map[string]*mssql.Server{},
		passwords:   map[string]string{},
		spids:       map[string]int{},
		queries:     map[string]*activeQuery{},
	}
}

func (m *Manager) Store() *storage.Store {
	return m.store
}

func (m *Manager) Profiles() ([]model.ConnectionProfile, error) {
	return m.store.LoadProfiles()
}

func (m *Manager) SaveProfile(req model.SaveConnectionRequest) (model.ConnectionProfile, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return model.ConnectionProfile{}, errors.New("connection name cannot be empty")
	}
	profiles, err := m.store.LoadProfiles()
	if err != nil {
		return model.ConnectionProfile{}, err
	}
	id := ""
	if req.ID != nil {
		id = strings.TrimSpace(*req.ID)
	}
	if id == "" {
		id = randomID()
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	for _, profile := range profiles {
		if profile.ID != id && strings.EqualFold(strings.TrimSpace(profile.Name), name) {
			return model.ConnectionProfile{}, fmt.Errorf(
				"a connection named '%s' already exists. Please choose a different name", name,
			)
		}
	}

	passwordChanged := req.Password != nil && *req.Password != ""
	if passwordChanged {
		if err := m.credentials.Set(id, *req.Password); err != nil {
			return model.ConnectionProfile{}, fmt.Errorf("store password in OS keychain: %w", err)
		}
		m.setCachedPassword(id, *req.Password)
	} else if req.CopyPasswordFrom != nil && *req.CopyPasswordFrom != "" {
		password, err := m.password(*req.CopyPasswordFrom)
		if err != nil {
			return model.ConnectionProfile{}, err
		}
		if password != "" {
			if err := m.credentials.Set(id, password); err != nil {
				return model.ConnectionProfile{}, fmt.Errorf("copy password into OS keychain: %w", err)
			}
			m.setCachedPassword(id, password)
		}
	}

	profile := model.ConnectionProfile{
		ConnectionConfig: req.ConnectionConfig,
		ID:               id,
		Engine:           "mssql",
		CreatedAt:        now,
		UpdatedAt:        now,
	}
	profile.Name = name
	if profile.Port <= 0 || profile.Port > 65535 {
		profile.Port = 1433
	}
	if profile.Database == "" {
		profile.Database = "master"
	}

	replaced := false
	for i := range profiles {
		if profiles[i].ID == id {
			profile.CreatedAt = profiles[i].CreatedAt
			profiles[i] = profile
			replaced = true
			break
		}
	}
	if !replaced {
		profiles = append(profiles, profile)
	}
	if err := m.store.SaveProfiles(profiles); err != nil {
		return model.ConnectionProfile{}, err
	}
	// An edited endpoint must never reuse the previous pool.
	m.disconnectOnly(id)
	return profile, nil
}

func (m *Manager) DeleteProfile(id string) error {
	profiles, err := m.store.LoadProfiles()
	if err != nil {
		return err
	}
	filtered := profiles[:0]
	for _, profile := range profiles {
		if profile.ID != id {
			filtered = append(filtered, profile)
		}
	}
	if err := m.store.SaveProfiles(filtered); err != nil {
		return err
	}
	m.disconnectOnly(id)
	m.mu.Lock()
	delete(m.passwords, id)
	delete(m.spids, id)
	m.mu.Unlock()
	return m.credentials.Delete(id)
}

func (m *Manager) TestConnection(req model.SaveConnectionRequest) error {
	password := ""
	if req.Password != nil {
		password = *req.Password
	} else if req.CopyPasswordFrom != nil {
		var err error
		password, err = m.password(*req.CopyPasswordFrom)
		if err != nil {
			return err
		}
	}
	profile := model.ConnectionProfile{
		ConnectionConfig: req.ConnectionConfig,
		ID:               "test",
		Engine:           "mssql",
	}
	if profile.Port == 0 {
		profile.Port = 1433
	}
	if profile.Database == "" {
		profile.Database = "master"
	}
	server, err := mssql.Open(profile, password)
	if err != nil {
		return err
	}
	defer server.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 25*time.Second)
	defer cancel()
	return server.Ping(ctx)
}

func (m *Manager) Connect(ctx context.Context, id string) error {
	if current := m.server(id); current != nil {
		probe, cancel := context.WithTimeout(ctx, 3*time.Second)
		err := current.Ping(probe)
		cancel()
		if err == nil {
			return nil
		}
		m.disconnectOnly(id)
	}
	profile, password, err := m.profileAndPassword(id)
	if err != nil {
		return err
	}
	server, err := mssql.Open(profile, password)
	if err != nil {
		return err
	}
	if err := server.Ping(ctx); err != nil {
		server.Close()
		return err
	}
	m.mu.Lock()
	if old := m.servers[id]; old != nil {
		_ = old.Close()
	}
	m.servers[id] = server
	m.mu.Unlock()
	return nil
}

func (m *Manager) Disconnect(id string) error {
	m.disconnectOnly(id)
	return nil
}

func (m *Manager) Server(ctx context.Context, id string) (*mssql.Server, error) {
	if server := m.server(id); server != nil {
		return server, nil
	}
	if err := m.Connect(ctx, id); err != nil {
		return nil, err
	}
	server := m.server(id)
	if server == nil {
		return nil, fmt.Errorf("not connected to connection '%s'", id)
	}
	return server, nil
}

func (m *Manager) ExecuteQuery(
	ctx context.Context,
	id string,
	database *string,
	statement string,
	maxRows *int,
	requestID *string,
	timeoutSeconds *int,
	onProgress func(spid int, request string),
) (model.QueryResult, error) {
	server, err := m.Server(ctx, id)
	if err != nil {
		return model.QueryResult{}, err
	}
	reqID := ""
	if requestID != nil {
		reqID = strings.TrimSpace(*requestID)
	}
	if reqID == "" {
		reqID = randomID()
	}
	queryCtx, cancel := context.WithCancel(ctx)
	state := &activeQuery{connectionID: id, cancel: cancel}
	m.mu.Lock()
	m.queries[reqID] = state
	m.mu.Unlock()
	defer func() {
		cancel()
		m.mu.Lock()
		delete(m.queries, reqID)
		m.mu.Unlock()
	}()

	result := server.Execute(queryCtx, database, statement, maxRows, timeoutSeconds, func(spid int) {
		m.mu.Lock()
		if current := m.queries[reqID]; current != nil {
			current.spid = spid
		}
		if spid > 0 {
			m.spids[id] = spid
		}
		m.mu.Unlock()
		if onProgress != nil {
			onProgress(spid, reqID)
		}
	})
	return result, nil
}

func (m *Manager) CancelQuery(connectionID string, requestID *string) error {
	var state *activeQuery
	m.mu.Lock()
	if requestID != nil && strings.TrimSpace(*requestID) != "" {
		state = m.queries[strings.TrimSpace(*requestID)]
		delete(m.queries, strings.TrimSpace(*requestID))
	} else {
		for key, candidate := range m.queries {
			if candidate.connectionID == connectionID {
				state = candidate
				delete(m.queries, key)
				break
			}
		}
	}
	m.mu.Unlock()
	if state == nil {
		m.disconnectOnly(connectionID)
		return nil
	}
	state.cancel()
	// A cancelled TDS stream must not return to the pool. Release it immediately, then use a
	// separate connection for KILL before returning. The next command lazily opens a fresh pool;
	// reconnecting in the background can otherwise race this KILL and target its replacement SPID.
	m.disconnectOnly(connectionID)
	if state.spid > 0 {
		m.killSPID(connectionID, state.spid)
	}
	return nil
}

func (m *Manager) SPID(id string) *int {
	m.mu.RLock()
	value, ok := m.spids[id]
	m.mu.RUnlock()
	if !ok {
		return nil
	}
	return &value
}

func (m *Manager) password(id string) (string, error) {
	m.mu.RLock()
	cached := m.passwords[id]
	m.mu.RUnlock()
	if cached != "" {
		return cached, nil
	}
	password, found, err := m.credentials.Get(id)
	if err != nil {
		return "", fmt.Errorf("retrieve password from OS keychain: %w", err)
	}
	if !found {
		return "", nil
	}
	m.setCachedPassword(id, password)
	return password, nil
}

func (m *Manager) profileAndPassword(id string) (model.ConnectionProfile, string, error) {
	profiles, err := m.store.LoadProfiles()
	if err != nil {
		return model.ConnectionProfile{}, "", err
	}
	for _, profile := range profiles {
		if profile.ID == id {
			password, err := m.password(id)
			return profile, password, err
		}
	}
	return model.ConnectionProfile{}, "", fmt.Errorf("connection profile %s not found", id)
}

func (m *Manager) server(id string) *mssql.Server {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.servers[id]
}

func (m *Manager) disconnectOnly(id string) {
	m.mu.Lock()
	server := m.servers[id]
	delete(m.servers, id)
	delete(m.spids, id)
	m.mu.Unlock()
	if server != nil {
		_ = server.Close()
	}
}

func (m *Manager) killSPID(connectionID string, spid int) {
	profile, password, err := m.profileAndPassword(connectionID)
	if err != nil {
		return
	}
	aux, err := mssql.Open(profile, password)
	if err != nil {
		return
	}
	defer aux.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, _ = aux.DB.ExecContext(ctx, fmt.Sprintf("KILL %d", spid))
}

func (m *Manager) setCachedPassword(id, password string) {
	m.mu.Lock()
	m.passwords[id] = password
	m.mu.Unlock()
}

func randomID() string {
	const alphabet = "0123456789abcdef"
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("conn-%d", time.Now().UnixNano())
	}
	for i := range bytes {
		bytes[i] = alphabet[int(bytes[i])%len(alphabet)]
	}
	return string(bytes[:8]) + "-" + string(bytes[8:12]) + "-" +
		string(bytes[12:16]) + "-" + string(bytes[16:20]) + "-" + string(bytes[20:])
}
