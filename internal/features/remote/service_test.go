package remote

import (
	"errors"
	"strings"
	"testing"
)

type fakeRepo struct {
	servers   map[string]RemoteServer // id -> server
	byAgent   map[string]RemoteServer // agent token hash -> server
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{servers: map[string]RemoteServer{}, byAgent: map[string]RemoteServer{}}
}

func (f *fakeRepo) ListRemoteServers() ([]RemoteServer, error) {
	var out []RemoteServer
	for _, s := range f.servers {
		out = append(out, s)
	}
	return out, nil
}
func (f *fakeRepo) CreateRemoteServer(input CreateServerInput, serverTokenHash string, agentTokenHash string) (RemoteServer, error) {
	s := RemoteServer{ID: "server-1", Name: input.Name, Host: input.Host, ConnectionMode: input.ConnectionMode, Status: "pending"}
	f.servers[s.ID] = s
	return s, nil
}
func (f *fakeRepo) UpdateRemoteServer(id string, input UpdateServerInput) (RemoteServer, error) {
	return RemoteServer{}, nil
}
func (f *fakeRepo) DeleteRemoteServer(id string) error { return nil }
func (f *fakeRepo) ListRemoteTasks(serverID string) ([]RemoteTask, error) {
	return nil, nil
}
func (f *fakeRepo) CreateRemoteTask(serverID string, input CreateTaskInput) (RemoteTask, error) {
	return RemoteTask{}, nil
}
func (f *fakeRepo) FindRemoteServerByAgentTokenHash(tokenHash string) (RemoteServer, error) {
	s, ok := f.byAgent[tokenHash]
	if !ok {
		return RemoteServer{}, errors.New("agent token not found")
	}
	return s, nil
}
func (f *fakeRepo) HeartbeatRemoteServer(id string, status string, metadata map[string]any) (RemoteServer, error) {
	return RemoteServer{}, nil
}
func (f *fakeRepo) ClaimNextRemoteTask(serverID string) (*RemoteTask, error) {
	return nil, nil
}
func (f *fakeRepo) UpdateRemoteTask(serverID string, taskID string, status string, outputText string) (RemoteTask, error) {
	return RemoteTask{}, nil
}
func (f *fakeRepo) ListRemoteTaskLogs(serverID string, taskID string) ([]TaskLog, error) {
	return nil, nil
}
func (f *fakeRepo) CreateRemoteTaskLog(serverID string, taskID string, eventKind string, message string) error {
	return nil
}
func (f *fakeRepo) ListAgentLogs(serverID string) ([]AgentLog, error) {
	return nil, nil
}
func (f *fakeRepo) CreateAgentLog(serverID string, input CreateAgentLogInput) (AgentLog, error) {
	return AgentLog{}, nil
}
func (f *fakeRepo) CreateTrafficSampleFromAgent(sampleScope string, scopeID string, rxBytes int64, txBytes int64, rate map[string]any) error {
	return nil
}

func TestCreateServerGeneratesTokens(t *testing.T) {
	repo := newFakeRepo()
	svc := NewService(repo)

	enrollment, err := svc.CreateServer(CreateServerInput{
		Name:           "Tokyo",
		Host:           "203.0.113.10",
		ConnectionMode: "pull",
	})
	if err != nil {
		t.Fatalf("CreateServer returned error: %v", err)
	}
	if !strings.HasPrefix(enrollment.ServerToken, "hxs_") {
		t.Fatalf("unexpected server token prefix: %q", enrollment.ServerToken)
	}
	if !strings.HasPrefix(enrollment.AgentToken, "hxa_") {
		t.Fatalf("unexpected agent token prefix: %q", enrollment.AgentToken)
	}
	if enrollment.ServerToken == enrollment.AgentToken {
		t.Fatal("server token and agent token must differ")
	}
}

func TestCreateServerValidatesInput(t *testing.T) {
	svc := NewService(newFakeRepo())
	if _, err := svc.CreateServer(CreateServerInput{Name: "", Host: "x", ConnectionMode: "pull"}); err == nil {
		t.Fatal("empty name accepted")
	}
	if _, err := svc.CreateServer(CreateServerInput{Name: "x", Host: "", ConnectionMode: "pull"}); err == nil {
		t.Fatal("empty host accepted")
	}
	if _, err := svc.CreateServer(CreateServerInput{Name: "x", Host: "y", ConnectionMode: "bogus"}); err == nil {
		t.Fatal("unsupported connection mode accepted")
	}
}

func TestAuthenticateAgent(t *testing.T) {
	repo := newFakeRepo()
	server := RemoteServer{ID: "server-1", Name: "Tokyo", Host: "203.0.113.10", Status: "pending"}
	repo.byAgent[hashToken("hxa_valid")] = server
	svc := NewService(repo)

	got, err := svc.authenticateAgent("hxa_valid")
	if err != nil {
		t.Fatalf("authenticateAgent returned error: %v", err)
	}
	if got.ID != server.ID {
		t.Fatalf("unexpected server: %+v", got)
	}

	if _, err := svc.authenticateAgent(""); err == nil {
		t.Fatal("empty token accepted")
	}
	if _, err := svc.authenticateAgent("hxa_invalid"); err == nil {
		t.Fatal("invalid token accepted")
	}
}

func TestHashTokenDeterministic(t *testing.T) {
	if hashToken("abc") != hashToken("abc") {
		t.Fatal("hashToken must be deterministic")
	}
	if hashToken("abc") == hashToken("abd") {
		t.Fatal("different tokens must hash differently")
	}
	if len(hashToken("abc")) == 0 {
		t.Fatal("hashToken returned empty")
	}
}

func TestSupportedTaskKinds(t *testing.T) {
	for _, kind := range []string{"install-xray", "restart-xray", "reload-config", "apply-xray-config", "render-xray-inbound", "collect-xray-stats", "apply-nginx-config", "issue-certificate", "sync-external-subscription", "run-vps-maintenance", "apply-security-policy", "run-notification-automation", "install-nginx", "renew-certificate", "install-warp", "shell-script"} {
		if !supportedTaskKind(kind) {
			t.Fatalf("expected supported task kind %q", kind)
		}
	}
	if supportedTaskKind("rm-rf") {
		t.Fatal("unsupported task kind accepted")
	}
}
