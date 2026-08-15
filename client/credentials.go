package client

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// ErrNotLoggedIn is returned when no credential is stored for a server.
var ErrNotLoggedIn = errors.New("not logged in: run `atkins --login <url>`")

// Credential is a stored login for one server.
type Credential struct {
	Server       string `json:"server"`
	UserID       string `json:"user_id"`
	Username     string `json:"username"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresAt    int64  `json:"expires_at"`
}

// Expired reports whether the access token is past, or nearly past, its
// expiry. The skew means a request never goes out with a token that
// expires while it's in flight.
func (c *Credential) Expired() bool {
	const skew = time.Minute
	if c == nil || c.ExpiresAt == 0 {
		return true
	}
	return time.Now().Add(skew).After(time.Unix(c.ExpiresAt, 0))
}

// Store is the on-disk credential file. It holds one credential per
// server so a machine can talk to more than one atkins instance, plus
// the server used when none is named.
type Store struct {
	Default string                 `json:"default"`
	Servers map[string]*Credential `json:"servers"`

	// path is where the store was loaded from and where Save writes.
	path string
}

// CredentialsPath returns the location of the credentials file.
//
// The configured path wins; ATKINS_CREDENTIALS reaches it through the
// config overlay. The environment is still read directly as a fallback
// so a command that never loaded configuration — a test, or a tool
// embedding this package — still honours it.
func CredentialsPath() (string, error) {
	if path := Settings().Credentials; path != "" {
		return path, nil
	}
	if path := os.Getenv("ATKINS_CREDENTIALS"); path != "" {
		return path, nil
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locate home directory: %w", err)
	}
	return filepath.Join(home, ".atkins", "credentials.json"), nil
}

// LoadStore reads the credential file. A missing file is not an error;
// it yields an empty store ready to be written to.
func LoadStore() (*Store, error) {
	path, err := CredentialsPath()
	if err != nil {
		return nil, err
	}

	store := &Store{
		Servers: map[string]*Credential{},
		path:    path,
	}

	contents, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return store, nil
		}
		return nil, fmt.Errorf("read credentials: %w", err)
	}

	if err := json.Unmarshal(contents, store); err != nil {
		return nil, fmt.Errorf("parse credentials %s: %w", path, err)
	}
	if store.Servers == nil {
		store.Servers = map[string]*Credential{}
	}
	store.path = path

	return store, nil
}

// Save writes the store back to disk.
//
// The file holds bearer tokens, so both it and its directory are
// created 0600/0700 and the write goes through a temporary file: a
// crash mid-write should not leave a truncated credential behind.
func (s *Store) Save() error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o700); err != nil {
		return fmt.Errorf("create credentials directory: %w", err)
	}

	contents, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	contents = append(contents, '\n')

	temp := s.path + ".tmp"
	if err := os.WriteFile(temp, contents, 0o600); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}
	if err := os.Rename(temp, s.path); err != nil {
		return fmt.Errorf("write credentials: %w", err)
	}

	return nil
}

// Get returns the credential for a server, or for the default server
// when server is empty.
func (s *Store) Get(server string) (*Credential, bool) {
	server = NormalizeServer(server)
	if server == "" {
		server = s.Default
	}
	if server == "" {
		return nil, false
	}

	credential, ok := s.Servers[server]
	return credential, ok
}

// Set stores a credential and makes its server the default.
func (s *Store) Set(credential *Credential) {
	if s.Servers == nil {
		s.Servers = map[string]*Credential{}
	}
	s.Servers[credential.Server] = credential
	s.Default = credential.Server
}

// Remove drops the credential for a server.
func (s *Store) Remove(server string) {
	server = NormalizeServer(server)
	delete(s.Servers, server)

	if s.Default == server {
		s.Default = ""
		for name := range s.Servers {
			s.Default = name
			break
		}
	}
}

// NormalizeServer trims a server URL to its canonical form so
// `https://ci.example.com/` and `https://ci.example.com` are one entry.
func NormalizeServer(server string) string {
	return strings.TrimRight(strings.TrimSpace(server), "/")
}
