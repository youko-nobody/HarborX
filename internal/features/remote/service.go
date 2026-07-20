package remote

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"strings"
)

type Summary struct {
	ConnectionModes []string `json:"connectionModes"`
	Capabilities    []string `json:"capabilities"`
}

type RemoteServer struct {
	ID             string         `json:"id"`
	Name           string         `json:"name"`
	Host           string         `json:"host"`
	ConnectionMode string         `json:"connectionMode"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type CreateServerInput struct {
	Name           string         `json:"name"`
	Host           string         `json:"host"`
	ConnectionMode string         `json:"connectionMode"`
	Metadata       map[string]any `json:"metadata"`
}

type UpdateServerInput struct {
	Name           string         `json:"name"`
	Host           string         `json:"host"`
	ConnectionMode string         `json:"connectionMode"`
	Status         string         `json:"status"`
	Metadata       map[string]any `json:"metadata"`
}

type ServerEnrollment struct {
	Server      RemoteServer `json:"server"`
	ServerToken string       `json:"serverToken"`
	AgentToken  string       `json:"agentToken"`
}

type RemoteTask struct {
	ID             string         `json:"id"`
	RemoteServerID string         `json:"remoteServerId"`
	TaskKind       string         `json:"taskKind"`
	Status         string         `json:"status"`
	Payload        map[string]any `json:"payload"`
	OutputText     string         `json:"outputText"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type CreateTaskInput struct {
	TaskKind string         `json:"taskKind"`
	Payload  map[string]any `json:"payload"`
}

type AgentHeartbeatInput struct {
	Status   string         `json:"status"`
	Metadata map[string]any `json:"metadata"`
}

type AgentTaskUpdateInput struct {
	Status     string `json:"status"`
	OutputText string `json:"outputText"`
}

type AgentTaskClaim struct {
	Server RemoteServer `json:"server"`
	Task   *RemoteTask  `json:"task"`
}

type Repository interface {
	ListRemoteServers() ([]RemoteServer, error)
	CreateRemoteServer(input CreateServerInput, serverTokenHash string, agentTokenHash string) (RemoteServer, error)
	UpdateRemoteServer(id string, input UpdateServerInput) (RemoteServer, error)
	DeleteRemoteServer(id string) error
	ListRemoteTasks(serverID string) ([]RemoteTask, error)
	CreateRemoteTask(serverID string, input CreateTaskInput) (RemoteTask, error)
	FindRemoteServerByAgentTokenHash(tokenHash string) (RemoteServer, error)
	HeartbeatRemoteServer(id string, status string, metadata map[string]any) (RemoteServer, error)
	ClaimNextRemoteTask(serverID string) (*RemoteTask, error)
	UpdateRemoteTask(serverID string, taskID string, status string, outputText string) (RemoteTask, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		ConnectionModes: []string{"websocket", "http", "pull"},
		Capabilities: []string{
			"server-registry",
			"agent-heartbeat",
			"streamed-install-tasks",
			"xray-manage",
			"nginx-manage",
			"warp-manage",
		},
	}
}

func (s Service) ListServers() ([]RemoteServer, error) {
	if s.repo == nil {
		return nil, errors.New("remote repository is not configured")
	}
	return s.repo.ListRemoteServers()
}

func (s Service) CreateServer(input CreateServerInput) (ServerEnrollment, error) {
	if s.repo == nil {
		return ServerEnrollment{}, errors.New("remote repository is not configured")
	}
	if err := validateServerInput(input.Name, input.Host, input.ConnectionMode); err != nil {
		return ServerEnrollment{}, err
	}
	serverToken, serverTokenHash, err := newToken("hxs")
	if err != nil {
		return ServerEnrollment{}, err
	}
	agentToken, agentTokenHash, err := newToken("hxa")
	if err != nil {
		return ServerEnrollment{}, err
	}
	server, err := s.repo.CreateRemoteServer(input, serverTokenHash, agentTokenHash)
	if err != nil {
		return ServerEnrollment{}, err
	}
	return ServerEnrollment{Server: server, ServerToken: serverToken, AgentToken: agentToken}, nil
}

func (s Service) UpdateServer(id string, input UpdateServerInput) (RemoteServer, error) {
	if s.repo == nil {
		return RemoteServer{}, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return RemoteServer{}, errors.New("remote server id is required")
	}
	if err := validateServerInput(input.Name, input.Host, input.ConnectionMode); err != nil {
		return RemoteServer{}, err
	}
	if strings.TrimSpace(input.Status) == "" {
		input.Status = "pending"
	}
	if !supportedStatus(input.Status) {
		return RemoteServer{}, errors.New("unsupported remote server status")
	}
	return s.repo.UpdateRemoteServer(id, input)
}

func (s Service) DeleteServer(id string) error {
	if s.repo == nil {
		return errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("remote server id is required")
	}
	return s.repo.DeleteRemoteServer(id)
}

func (s Service) ListTasks(serverID string) ([]RemoteTask, error) {
	if s.repo == nil {
		return nil, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(serverID) == "" {
		return nil, errors.New("remote server id is required")
	}
	return s.repo.ListRemoteTasks(serverID)
}

func (s Service) CreateTask(serverID string, input CreateTaskInput) (RemoteTask, error) {
	if s.repo == nil {
		return RemoteTask{}, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(serverID) == "" {
		return RemoteTask{}, errors.New("remote server id is required")
	}
	if strings.TrimSpace(input.TaskKind) == "" {
		return RemoteTask{}, errors.New("remote task kind is required")
	}
	if !supportedTaskKind(input.TaskKind) {
		return RemoteTask{}, errors.New("unsupported remote task kind")
	}
	return s.repo.CreateRemoteTask(serverID, input)
}

func (s Service) AgentHeartbeat(token string, input AgentHeartbeatInput) (RemoteServer, error) {
	server, err := s.authenticateAgent(token)
	if err != nil {
		return RemoteServer{}, err
	}
	status := strings.TrimSpace(input.Status)
	if status == "" {
		status = "online"
	}
	if !supportedStatus(status) {
		return RemoteServer{}, errors.New("unsupported remote server status")
	}
	return s.repo.HeartbeatRemoteServer(server.ID, status, input.Metadata)
}

func (s Service) AgentClaimTask(token string) (AgentTaskClaim, error) {
	server, err := s.authenticateAgent(token)
	if err != nil {
		return AgentTaskClaim{}, err
	}
	task, err := s.repo.ClaimNextRemoteTask(server.ID)
	if err != nil {
		return AgentTaskClaim{}, err
	}
	return AgentTaskClaim{Server: server, Task: task}, nil
}

func (s Service) AgentUpdateTask(token string, taskID string, input AgentTaskUpdateInput) (RemoteTask, error) {
	server, err := s.authenticateAgent(token)
	if err != nil {
		return RemoteTask{}, err
	}
	if strings.TrimSpace(taskID) == "" {
		return RemoteTask{}, errors.New("remote task id is required")
	}
	if !supportedTaskStatus(input.Status) {
		return RemoteTask{}, errors.New("unsupported remote task status")
	}
	return s.repo.UpdateRemoteTask(server.ID, taskID, input.Status, input.OutputText)
}

func (s Service) authenticateAgent(token string) (RemoteServer, error) {
	if s.repo == nil {
		return RemoteServer{}, errors.New("remote repository is not configured")
	}
	token = strings.TrimSpace(token)
	if token == "" {
		return RemoteServer{}, errors.New("missing agent token")
	}
	return s.repo.FindRemoteServerByAgentTokenHash(hashToken(token))
}

func validateServerInput(name string, host string, connectionMode string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("remote server name is required")
	}
	if strings.TrimSpace(host) == "" {
		return errors.New("remote server host is required")
	}
	if strings.TrimSpace(connectionMode) == "" {
		return errors.New("remote server connection mode is required")
	}
	if !supportedConnectionMode(connectionMode) {
		return errors.New("unsupported remote server connection mode")
	}
	return nil
}

func supportedConnectionMode(value string) bool {
	switch value {
	case "websocket", "http", "pull":
		return true
	default:
		return false
	}
}

func supportedStatus(value string) bool {
	switch value {
	case "pending", "online", "offline", "maintenance", "disabled":
		return true
	default:
		return false
	}
}

func supportedTaskKind(value string) bool {
	switch value {
	case "install-xray", "restart-xray", "reload-config", "install-nginx", "renew-certificate", "install-warp", "shell-script":
		return true
	default:
		return false
	}
}

func supportedTaskStatus(value string) bool {
	switch value {
	case "queued", "running", "succeeded", "failed", "cancelled":
		return true
	default:
		return false
	}
}

func newToken(prefix string) (string, string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", "", err
	}
	token := fmt.Sprintf("%s_%s", prefix, base64.RawURLEncoding.EncodeToString(raw))
	return token, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawStdEncoding.EncodeToString(sum[:])
}

func SecureCompareTokenHash(left string, right string) bool {
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
