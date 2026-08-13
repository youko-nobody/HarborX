package auth

import (
	"errors"
	"strings"
	"testing"
	"time"
)

type fakeRepo struct {
	users           map[string]User
	tokenHashes     map[string]User
	created         int
	tokensPrunedByUser map[string]int
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{
		users:              map[string]User{},
		tokenHashes:        map[string]User{},
		tokensPrunedByUser: map[string]int{},
	}
}

func (f *fakeRepo) GetUserByUsername(username string) (User, error) {
	for _, u := range f.users {
		if u.Username == username {
			return u, nil
		}
	}
	return User{}, errors.New("user not found")
}

func (f *fakeRepo) UpdateUserPasswordHash(userID string, passwordHash string) error {
	u, ok := f.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	u.PasswordHash = passwordHash
	f.users[userID] = u
	return nil
}

func (f *fakeRepo) CreateAPIToken(userID string, name string, tokenHash string) error {
	u, ok := f.users[userID]
	if !ok {
		return errors.New("user not found")
	}
	f.created++
	f.tokenHashes[tokenHash] = u
	return nil
}

func (f *fakeRepo) FindAPITokenByHash(tokenHash string) (User, error) {
	u, ok := f.tokenHashes[tokenHash]
	if !ok {
		return User{}, errors.New("token not found")
	}
	return u, nil
}

func (f *fakeRepo) DeleteAPITokensBefore(userID string, cutoff time.Time, maxSessions int) error {
	if userID == "" {
		return errors.New("user id is required")
	}
	f.tokensPrunedByUser[userID]++
	return nil
}

func seedActiveUser(repo *fakeRepo, username string, password string) (User, error) {
	hash, err := HashPassword(password)
	if err != nil {
		return User{}, err
	}
	u := User{ID: "user-1", Username: username, PasswordHash: hash, Role: "admin", Status: "active", DisplayName: "Admin"}
	repo.users[u.ID] = u
	return u, nil
}

func TestHashAndVerifyPassword(t *testing.T) {
	hash, err := HashPassword("correct horse battery staple")
	if err != nil {
		t.Fatalf("HashPassword returned error: %v", err)
	}
	if hash == "" || !strings.HasPrefix(hash, "pbkdf2$") {
		t.Fatalf("unexpected hash format: %q", hash)
	}
	if !VerifyPassword("correct horse battery staple", hash) {
		t.Fatal("VerifyPassword rejected the correct password")
	}
	if VerifyPassword("wrong password", hash) {
		t.Fatal("VerifyPassword accepted a wrong password")
	}
	// Corrupted hash must not panic and must fail.
	if VerifyPassword("x", "not-a-valid-hash") {
		t.Fatal("VerifyPassword accepted a malformed hash")
	}
}

func TestLoginSuccess(t *testing.T) {
	repo := newFakeRepo()
	if _, err := seedActiveUser(repo, "admin", "secret"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := NewService(repo)

	resp, err := svc.Login(LoginInput{Username: "admin", Password: "secret"})
	if err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if resp.Token == "" || !strings.HasPrefix(resp.Token, "hx_") {
		t.Fatalf("unexpected token: %q", resp.Token)
	}
	if resp.User.PasswordHash != "" {
		t.Fatal("Login response leaked password hash")
	}
	if resp.User.Username != "admin" {
		t.Fatalf("unexpected user: %+v", resp.User)
	}
	if repo.created != 1 {
		t.Fatalf("expected 1 api token created, got %d", repo.created)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	repo := newFakeRepo()
	if _, err := seedActiveUser(repo, "admin", "secret"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := NewService(repo)

	if _, err := svc.Login(LoginInput{Username: "admin", Password: "wrong"}); err == nil {
		t.Fatal("Login succeeded with wrong password")
	}
	if repo.created != 0 {
		t.Fatalf("expected 0 api tokens after failed login, got %d", repo.created)
	}
}

func TestLoginDisabledUser(t *testing.T) {
	repo := newFakeRepo()
	hash, _ := HashPassword("secret")
	repo.users["user-2"] = User{ID: "user-2", Username: "bob", PasswordHash: hash, Role: "member", Status: "disabled"}
	svc := NewService(repo)

	if _, err := svc.Login(LoginInput{Username: "bob", Password: "secret"}); err == nil {
		t.Fatal("Login succeeded for a disabled user")
	}
}

func TestLoginUnknownUser(t *testing.T) {
	svc := NewService(newFakeRepo())
	if _, err := svc.Login(LoginInput{Username: "nobody", Password: "secret"}); err == nil {
		t.Fatal("Login succeeded for an unknown user")
	}
}

func TestAuthenticateBearer(t *testing.T) {
	repo := newFakeRepo()
	u, err := seedActiveUser(repo, "admin", "secret")
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := NewService(repo)

	token := "hx_testtoken"
	repo.tokenHashes[HashToken(token)] = u

	got, err := svc.AuthenticateBearer("Bearer " + token)
	if err != nil {
		t.Fatalf("AuthenticateBearer returned error: %v", err)
	}
	if got.ID != u.ID {
		t.Fatalf("unexpected user: %+v", got)
	}
	if got.PasswordHash != "" {
		t.Fatal("AuthenticateBearer leaked password hash")
	}

	if _, err := svc.AuthenticateBearer(""); err == nil {
		t.Fatal("empty header accepted")
	}
	if _, err := svc.AuthenticateBearer("Token abc"); err == nil {
		t.Fatal("non-bearer header accepted")
	}
	if _, err := svc.AuthenticateBearer("Bearer invalid-token"); err == nil {
		t.Fatal("invalid token accepted")
	}
}

func TestBootstrapAdminPassword(t *testing.T) {
	repo := newFakeRepo()
	// Admin user exists but has an empty password hash; the service must
	// bootstrap a password hash on construction.
	repo.users["user-admin"] = User{ID: "user-admin", Username: "admin", PasswordHash: "", Role: "admin", Status: "active"}
	NewService(repo)

	admin, err := repo.GetUserByUsername("admin")
	if err != nil {
		t.Fatalf("admin missing: %v", err)
	}
	if admin.PasswordHash == "" {
		t.Fatal("bootstrap admin password hash was not set")
	}
	if !strings.HasPrefix(admin.PasswordHash, "pbkdf2$") {
		t.Fatalf("unexpected hash format: %q", admin.PasswordHash)
	}
}

func TestLoginPrunesOldSessions(t *testing.T) {
	repo := newFakeRepo()
	if _, err := seedActiveUser(repo, "admin", "secret"); err != nil {
		t.Fatalf("seed: %v", err)
	}
	svc := NewService(repo)

	if _, err := svc.Login(LoginInput{Username: "admin", Password: "secret"}); err != nil {
		t.Fatalf("Login returned error: %v", err)
	}
	if repo.tokensPrunedByUser["user-1"] != 1 {
		t.Fatalf("expected session pruning to run, got count=%d", repo.tokensPrunedByUser["user-1"])
	}
}

func TestDeleteAPITokensBeforeCutoff(t *testing.T) {
	t.Skip("exercised indirectly by LoginPrunesOldSessions; the sqlite implementation is covered by the storage layer")
	_ = time.Now()
}
