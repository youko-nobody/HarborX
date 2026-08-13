package users

import (
	"errors"
	"strings"
	"testing"
)

type fakeUserRepo struct {
	users   map[string]User
	created []User
}

func newFakeUserRepo() *fakeUserRepo {
	return &fakeUserRepo{users: map[string]User{}}
}

func (f *fakeUserRepo) ListUsers() ([]User, error) {
	var items []User
	for _, u := range f.users {
		items = append(items, u)
	}
	return items, nil
}

func (f *fakeUserRepo) CreateUser(input CreateInput, passwordHash string) (User, error) {
	item := User{ID: "user-" + input.Username, Username: input.Username, Role: input.Role}
	f.users[item.ID] = item
	f.created = append(f.created, item)
	return item, nil
}

func (f *fakeUserRepo) UpdateUser(id string, input UpdateInput, passwordHash string) (User, error) {
	u, ok := f.users[id]
	if !ok {
		return User{}, errors.New("user not found")
	}
	if input.Role != "" {
		u.Role = input.Role
	}
	if input.Status != "" {
		u.Status = input.Status
	}
	f.users[id] = u
	return u, nil
}

func (f *fakeUserRepo) DeleteUser(id string) error {
	if _, ok := f.users[id]; !ok {
		return errors.New("user not found")
	}
	delete(f.users, id)
	return nil
}

func TestCreateRequiresExplicitRole(t *testing.T) {
	svc := NewService(newFakeUserRepo())
	_, err := svc.Create(CreateInput{Username: "a", Password: "p", Role: ""})
	if err == nil || !strings.Contains(err.Error(), "role is required") {
		t.Fatalf("empty role was accepted: %v", err)
	}
}

func TestUpdateDoesNotSilentDowngradeAdmin(t *testing.T) {
	repo := newFakeUserRepo()
	repo.users["user-admin"] = User{ID: "user-admin", Username: "admin", Role: "admin", Status: "active"}
	svc := NewService(repo)

	// Changing only the password while leaving role empty must be rejected so
	// the admin cannot accidentally be downgraded to "member".
	_, err := svc.Update("user-admin", UpdateInput{Password: "new-pass"})
	if err == nil || !strings.Contains(err.Error(), "role is required") {
		t.Fatalf("role-less update silently applied: %v; user=%+v", err, repo.users["user-admin"])
	}
	if repo.users["user-admin"].Role != "admin" {
		t.Fatalf("admin was downgraded: %s", repo.users["user-admin"].Role)
	}
}

func TestUpdateRequiresExplicitStatus(t *testing.T) {
	repo := newFakeUserRepo()
	repo.users["u"] = User{ID: "u", Username: "u", Role: "member", Status: "disabled"}
	svc := NewService(repo)

	_, err := svc.Update("u", UpdateInput{Role: "member"})
	if err == nil || !strings.Contains(err.Error(), "status is required") {
		t.Fatalf("status-less update silently applied: %v", err)
	}
}

func TestUpdatePreservesExistingRoleWhenExplicit(t *testing.T) {
	repo := newFakeUserRepo()
	repo.users["u"] = User{ID: "u", Username: "u", Role: "admin", Status: "active"}
	svc := NewService(repo)

	updated, err := svc.Update("u", UpdateInput{Role: "admin", Status: "active", Password: "new"})
	if err != nil {
		t.Fatalf("update returned error: %v", err)
	}
	if updated.Role != "admin" {
		t.Fatalf("role changed unexpectedly: %s", updated.Role)
	}
}

func TestCreateMemberRole(t *testing.T) {
	repo := newFakeUserRepo()
	svc := NewService(repo)
	created, err := svc.Create(CreateInput{Username: "m", Password: "p", Role: "member"})
	if err != nil {
		t.Fatalf("create returned error: %v", err)
	}
	if created.Role != "member" {
		t.Fatalf("unexpected role: %s", created.Role)
	}
}