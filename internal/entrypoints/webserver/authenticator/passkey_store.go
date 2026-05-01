package authenticator

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	passkeylib "github.com/egregors/passkey"
	"github.com/go-webauthn/webauthn/webauthn"
)

const fileModePrivate = 0o600

type passkeySessionData interface {
	webauthn.SessionData | passkeylib.UserSessionData
}

type passkeySessionsSnapshot[T passkeySessionData] struct {
	Sessions map[string]T `json:"sessions"`
}

type passkeySessionStore[T passkeySessionData] struct {
	mu       sync.RWMutex
	path     string
	sessions map[string]T
}

func newPasskeySessionStore[T passkeySessionData](path string) (*passkeySessionStore[T], error) {
	store := &passkeySessionStore[T]{
		path:     path,
		sessions: map[string]T{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *passkeySessionStore[T]) Create(data T) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	sessionID, err := generatePasskeyToken()
	if err != nil {
		return "", err
	}

	s.sessions[sessionID] = data
	if err = s.flushLocked(); err != nil {
		delete(s.sessions, sessionID)
		return "", err
	}

	return sessionID, nil
}

func (s *passkeySessionStore[T]) Delete(token string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.sessions[token]; !ok {
		return
	}

	delete(s.sessions, token)
	_ = s.flushLocked()
}

func (s *passkeySessionStore[T]) Get(token string) (*T, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	value, ok := s.sessions[token]
	if !ok {
		return nil, false
	}

	copyValue := value
	return &copyValue, true
}

func (s *passkeySessionStore[T]) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create passkey sessions dir: %w", err)
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read passkey sessions file: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}

	var snapshot passkeySessionsSnapshot[T]
	if err = json.Unmarshal(payload, &snapshot); err != nil {
		return fmt.Errorf("decode passkey sessions file: %w", err)
	}
	if snapshot.Sessions == nil {
		snapshot.Sessions = map[string]T{}
	}

	s.sessions = snapshot.Sessions
	return nil
}

func (s *passkeySessionStore[T]) flushLocked() error {
	payload, err := json.Marshal(passkeySessionsSnapshot[T]{Sessions: s.sessions})
	if err != nil {
		return fmt.Errorf("encode passkey sessions file: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp", s.path)
	if err = os.WriteFile(tmpPath, payload, fileModePrivate); err != nil {
		return fmt.Errorf("write passkey sessions temp file: %w", err)
	}
	if err = os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace passkey sessions file: %w", err)
	}

	return nil
}

type passkeyUsersSnapshot struct {
	Users []passkeyUser `json:"users"`
}

type passkeyUserStore struct {
	mu     sync.RWMutex
	path   string
	byName map[string]*passkeyUser
	byID   map[string]string
}

func newPasskeyUserStore(path string) (*passkeyUserStore, error) {
	store := &passkeyUserStore{
		path:   path,
		byName: map[string]*passkeyUser{},
		byID:   map[string]string{},
	}

	if err := store.load(); err != nil {
		return nil, err
	}

	return store, nil
}

func (s *passkeyUserStore) Create(username string) (passkeylib.User, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	username = strings.TrimSpace(username)
	if username == "" {
		return nil, errors.New("username is empty")
	}

	if _, exists := s.byName[username]; exists {
		return nil, fmt.Errorf("user %s already exists", username)
	}

	userID, err := generatePasskeyUserID()
	if err != nil {
		return nil, err
	}

	user := &passkeyUser{
		ID:          userID,
		Name:        username,
		DisplayName: username,
	}

	s.byName[username] = user.clone()
	s.byID[passkeyUserIDKey(userID)] = username
	if err = s.flushLocked(); err != nil {
		delete(s.byName, username)
		delete(s.byID, passkeyUserIDKey(userID))
		return nil, err
	}

	return user.clone(), nil
}

func (s *passkeyUserStore) Update(user passkeylib.User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, ok := user.(*passkeyUser)
	if !ok {
		return fmt.Errorf("unsupported passkey user type %T", user)
	}

	if strings.TrimSpace(record.Name) == "" {
		return errors.New("username is empty")
	}
	if len(record.ID) == 0 {
		return errors.New("user id is empty")
	}
	if strings.TrimSpace(record.DisplayName) == "" {
		record.DisplayName = record.Name
	}

	idKey := passkeyUserIDKey(record.ID)
	if existingName, exists := s.byID[idKey]; exists && existingName != record.Name {
		return fmt.Errorf("user id collision for %s", record.Name)
	}

	s.byName[record.Name] = record.clone()
	s.byID[idKey] = record.Name

	if err := s.flushLocked(); err != nil {
		return err
	}

	return nil
}

func (s *passkeyUserStore) Get(userID []byte) (passkeylib.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	username, exists := s.byID[passkeyUserIDKey(userID)]
	if !exists {
		return nil, errors.New("user not found")
	}

	user, exists := s.byName[username]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user.clone(), nil
}

func (s *passkeyUserStore) GetByName(username string) (passkeylib.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.byName[strings.TrimSpace(username)]
	if !exists {
		return nil, errors.New("user not found")
	}

	return user.clone(), nil
}

func (s *passkeyUserStore) load() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return fmt.Errorf("create passkey users dir: %w", err)
	}

	payload, err := os.ReadFile(s.path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil
		}
		return fmt.Errorf("read passkey users file: %w", err)
	}
	if len(payload) == 0 {
		return nil
	}

	var snapshot passkeyUsersSnapshot
	if err = json.Unmarshal(payload, &snapshot); err != nil {
		return fmt.Errorf("decode passkey users file: %w", err)
	}

	byName := make(map[string]*passkeyUser, len(snapshot.Users))
	byID := make(map[string]string, len(snapshot.Users))
	for _, user := range snapshot.Users {
		user.Name = strings.TrimSpace(user.Name)
		user.DisplayName = strings.TrimSpace(user.DisplayName)
		if user.Name == "" || len(user.ID) == 0 {
			continue
		}
		if user.DisplayName == "" {
			user.DisplayName = user.Name
		}

		clonedUser := user.clone()
		byName[user.Name] = clonedUser
		byID[passkeyUserIDKey(clonedUser.ID)] = clonedUser.Name
	}

	s.byName = byName
	s.byID = byID
	return nil
}

func (s *passkeyUserStore) flushLocked() error {
	users := make([]passkeyUser, 0, len(s.byName))
	for _, user := range s.byName {
		users = append(users, *user.clone())
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].Name < users[j].Name
	})

	payload, err := json.Marshal(passkeyUsersSnapshot{Users: users})
	if err != nil {
		return fmt.Errorf("encode passkey users file: %w", err)
	}

	tmpPath := fmt.Sprintf("%s.tmp", s.path)
	if err = os.WriteFile(tmpPath, payload, fileModePrivate); err != nil {
		return fmt.Errorf("write passkey users temp file: %w", err)
	}
	if err = os.Rename(tmpPath, s.path); err != nil {
		return fmt.Errorf("replace passkey users file: %w", err)
	}

	return nil
}

type passkeyUser struct {
	ID          []byte                `json:"id"`
	Name        string                `json:"name"`
	DisplayName string                `json:"displayName"`
	Credentials []webauthn.Credential `json:"credentials"`
}

func (u *passkeyUser) WebAuthnID() []byte {
	return cloneBytes(u.ID)
}

func (u *passkeyUser) WebAuthnName() string {
	return u.Name
}

func (u *passkeyUser) WebAuthnDisplayName() string {
	return u.DisplayName
}

func (u *passkeyUser) WebAuthnCredentials() []webauthn.Credential {
	return cloneCredentials(u.Credentials)
}

func (u *passkeyUser) PutCredential(credential webauthn.Credential) {
	for i, existing := range u.Credentials {
		if bytes.Equal(existing.ID, credential.ID) {
			u.Credentials[i] = cloneCredential(credential)
			return
		}
	}

	u.Credentials = append(u.Credentials, cloneCredential(credential))
}

func (u *passkeyUser) clone() *passkeyUser {
	if u == nil {
		return nil
	}

	return &passkeyUser{
		ID:          cloneBytes(u.ID),
		Name:        u.Name,
		DisplayName: u.DisplayName,
		Credentials: cloneCredentials(u.Credentials),
	}
}

func cloneCredentials(in []webauthn.Credential) []webauthn.Credential {
	if len(in) == 0 {
		return nil
	}

	out := make([]webauthn.Credential, len(in))
	for i, credential := range in {
		out[i] = cloneCredential(credential)
	}

	return out
}

func cloneCredential(in webauthn.Credential) webauthn.Credential {
	out := in
	out.ID = cloneBytes(in.ID)
	out.PublicKey = cloneBytes(in.PublicKey)
	out.Transport = append(out.Transport[:0:0], in.Transport...)
	out.Authenticator.AAGUID = cloneBytes(in.Authenticator.AAGUID)
	out.Attestation.ClientDataJSON = cloneBytes(in.Attestation.ClientDataJSON)
	out.Attestation.ClientDataHash = cloneBytes(in.Attestation.ClientDataHash)
	out.Attestation.AuthenticatorData = cloneBytes(in.Attestation.AuthenticatorData)
	out.Attestation.Object = cloneBytes(in.Attestation.Object)
	return out
}

func cloneBytes(in []byte) []byte {
	if len(in) == 0 {
		return nil
	}

	out := make([]byte, len(in))
	copy(out, in)
	return out
}

func passkeyUserIDKey(userID []byte) string {
	return base64.RawURLEncoding.EncodeToString(userID)
}

func generatePasskeyToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate passkey token: %w", err)
	}

	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func generatePasskeyUserID() ([]byte, error) {
	userID := make([]byte, 32)
	if _, err := rand.Read(userID); err != nil {
		return nil, fmt.Errorf("generate passkey user id: %w", err)
	}

	return userID, nil
}
