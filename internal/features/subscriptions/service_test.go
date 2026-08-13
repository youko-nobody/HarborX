package subscriptions

import (
	"errors"
	"testing"

	"harborx/internal/features/nodes"
	"harborx/internal/features/rules"
	"harborx/internal/features/templates"
)

type fakeRepo struct {
	tokens map[string]string
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{tokens: map[string]string{}}
}

func (f *fakeRepo) ListSubscriptions() ([]Subscription, error) { return nil, nil }
func (f *fakeRepo) CreateSubscription(input CreateInput) (Subscription, error) {
	return Subscription{}, nil
}
func (f *fakeRepo) UpdateSubscription(id string, input CreateInput) (Subscription, error) {
	return Subscription{}, nil
}
func (f *fakeRepo) DeleteSubscription(id string) error { return nil }
func (f *fakeRepo) GetSubscriptionAccessToken(id string) (string, error) {
	token, ok := f.tokens[id]
	if !ok {
		return "", errors.New("subscription not found")
	}
	return token, nil
}
func (f *fakeRepo) RotateSubscriptionAccessToken(id string) (string, error) {
	if _, ok := f.tokens[id]; !ok {
		return "", errors.New("subscription not found")
	}
	newToken := "rotated-token"
	f.tokens[id] = newToken
	return newToken, nil
}
func (f *fakeRepo) ListNodes() ([]nodes.Node, error)     { return nil, nil }
func (f *fakeRepo) ListRuleSets() ([]rules.RuleSet, error) { return nil, nil }
func (f *fakeRepo) ListTemplates() ([]templates.Template, error) {
	return nil, nil
}

func TestCheckAccess(t *testing.T) {
	repo := newFakeRepo()
	repo.tokens["sub-1"] = "sekrit-token"
	svc := NewService(repo)

	cases := []struct {
		name    string
		id      string
		token   string
		want    bool
	}{
		{"correct token", "sub-1", "sekrit-token", true},
		{"wrong token", "sub-1", "wrong", false},
		{"empty token", "sub-1", "", false},
		{"empty id", "", "sekrit-token", false},
		{"unknown subscription", "sub-2", "sekrit-token", false},
		{"empty stored token denies", "sub-3", "any", false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := svc.CheckAccess(tc.id, tc.token); got != tc.want {
				t.Fatalf("CheckAccess(%q, %q) = %v, want %v", tc.id, tc.token, got, tc.want)
			}
		})
	}
}

func TestCheckAccessNilRepo(t *testing.T) {
	var svc Service
	if svc.CheckAccess("sub-1", "token") {
		t.Fatal("nil repo must deny access")
	}
}

func TestRotateAccessToken(t *testing.T) {
	repo := newFakeRepo()
	repo.tokens["sub-1"] = "old-token"
	svc := NewService(repo)

	rotated, err := svc.RotateAccessToken("sub-1")
	if err != nil {
		t.Fatalf("RotateAccessToken returned error: %v", err)
	}
	if rotated.AccessToken != "rotated-token" {
		t.Fatalf("unexpected rotated token: %q", rotated.AccessToken)
	}
	if rotated.SubscriptionID != "sub-1" {
		t.Fatalf("unexpected subscription id: %q", rotated.SubscriptionID)
	}
	if rotated.RotatedAt == "" {
		t.Fatal("rotatedAt missing")
	}
	// The old token is no longer stored; rotation is a revocation.
	if repo.tokens["sub-1"] != "rotated-token" {
		t.Fatalf("stored token was not replaced: %q", repo.tokens["sub-1"])
	}
}

func TestRotateAccessTokenMissing(t *testing.T) {
	svc := NewService(newFakeRepo())
	_, err := svc.RotateAccessToken("nope")
	if err == nil {
		t.Fatal("expected error for unknown subscription")
	}
}
