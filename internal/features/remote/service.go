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

type TaskLog struct {
	ID             string `json:"id"`
	RemoteTaskID   string `json:"remoteTaskId"`
	RemoteServerID string `json:"remoteServerId"`
	EventKind      string `json:"eventKind"`
	Message        string `json:"message"`
	CreatedAt      string `json:"createdAt"`
}

type AgentLog struct {
	ID             string         `json:"id"`
	RemoteServerID string         `json:"remoteServerId"`
	Level          string         `json:"level"`
	Message        string         `json:"message"`
	Metadata       map[string]any `json:"metadata"`
	CreatedAt      string         `json:"createdAt"`
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

type CreateAgentLogInput struct {
	Level    string         `json:"level"`
	Message  string         `json:"message"`
	Metadata map[string]any `json:"metadata"`
}

type AgentTrafficSampleInput struct {
	SampleScope string         `json:"sampleScope"`
	ScopeID     string         `json:"scopeId"`
	RXBytes     int64          `json:"rxBytes"`
	TXBytes     int64          `json:"txBytes"`
	Rate        map[string]any `json:"rate"`
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
	ListRemoteTaskLogs(serverID string, taskID string) ([]TaskLog, error)
	CreateRemoteTaskLog(serverID string, taskID string, eventKind string, message string) error
	ListAgentLogs(serverID string) ([]AgentLog, error)
	CreateAgentLog(serverID string, input CreateAgentLogInput) (AgentLog, error)
	CreateTrafficSampleFromAgent(sampleScope string, scopeID string, rxBytes int64, txBytes int64, rate map[string]any) error
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
	task, err := s.repo.CreateRemoteTask(serverID, input)
	if err != nil {
		return RemoteTask{}, err
	}
	_ = s.repo.CreateRemoteTaskLog(serverID, task.ID, "queued", "task queued by operator")
	return task, nil
}

func (s Service) ListTaskLogs(serverID string, taskID string) ([]TaskLog, error) {
	if s.repo == nil {
		return nil, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(serverID) == "" {
		return nil, errors.New("remote server id is required")
	}
	return s.repo.ListRemoteTaskLogs(serverID, taskID)
}

func (s Service) ListAgentLogs(serverID string) ([]AgentLog, error) {
	if s.repo == nil {
		return nil, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(serverID) == "" {
		return nil, errors.New("remote server id is required")
	}
	return s.repo.ListAgentLogs(serverID)
}

func (s Service) CreateAgentLog(serverID string, input CreateAgentLogInput) (AgentLog, error) {
	if s.repo == nil {
		return AgentLog{}, errors.New("remote repository is not configured")
	}
	if strings.TrimSpace(serverID) == "" {
		return AgentLog{}, errors.New("remote server id is required")
	}
	if strings.TrimSpace(input.Level) == "" {
		input.Level = "info"
	}
	if strings.TrimSpace(input.Message) == "" {
		return AgentLog{}, errors.New("agent log message is required")
	}
	return s.repo.CreateAgentLog(serverID, input)
}

func (s Service) AgentLog(token string, input CreateAgentLogInput) (AgentLog, error) {
	server, err := s.authenticateAgent(token)
	if err != nil {
		return AgentLog{}, err
	}
	if strings.TrimSpace(input.Level) == "" {
		input.Level = "info"
	}
	if strings.TrimSpace(input.Message) == "" {
		return AgentLog{}, errors.New("agent log message is required")
	}
	return s.repo.CreateAgentLog(server.ID, input)
}

func (s Service) AgentTrafficSample(token string, input AgentTrafficSampleInput) error {
	server, err := s.authenticateAgent(token)
	if err != nil {
		return err
	}
	if strings.TrimSpace(input.SampleScope) == "" {
		input.SampleScope = "server"
	}
	if strings.TrimSpace(input.ScopeID) == "" {
		input.ScopeID = server.ID
	}
	if input.RXBytes < 0 || input.TXBytes < 0 {
		return errors.New("traffic bytes cannot be negative")
	}
	if input.Rate == nil {
		input.Rate = map[string]any{}
	}
	input.Rate["remoteServerId"] = server.ID
	return s.repo.CreateTrafficSampleFromAgent(input.SampleScope, input.ScopeID, input.RXBytes, input.TXBytes, input.Rate)
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
	updated, err := s.repo.HeartbeatRemoteServer(server.ID, status, input.Metadata)
	if err != nil {
		return RemoteServer{}, err
	}
	_, _ = s.repo.CreateAgentLog(server.ID, CreateAgentLogInput{
		Level:    "info",
		Message:  "agent heartbeat: " + status,
		Metadata: input.Metadata,
	})
	return updated, nil
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
	if task != nil {
		_ = s.repo.CreateRemoteTaskLog(server.ID, task.ID, "claimed", "task claimed by agent")
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
	task, err := s.repo.UpdateRemoteTask(server.ID, taskID, input.Status, input.OutputText)
	if err != nil {
		return RemoteTask{}, err
	}
	_ = s.repo.CreateRemoteTaskLog(server.ID, taskID, input.Status, input.OutputText)
	_, _ = s.repo.CreateAgentLog(server.ID, CreateAgentLogInput{
		Level:   logLevelForTaskStatus(input.Status),
		Message: "task " + taskID + " " + input.Status,
		Metadata: map[string]any{
			"taskKind": task.TaskKind,
			"taskId":   task.ID,
		},
	})
	return task, nil
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
	case "install-xray", "restart-xray", "reload-config", "apply-xray-config", "render-xray-inbound", "collect-xray-stats", "apply-nginx-config", "issue-certificate", "sync-external-subscription", "run-vps-maintenance", "apply-security-policy", "run-notification-automation", "install-nginx", "renew-certificate", "install-warp", "shell-script":
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

func logLevelForTaskStatus(status string) string {
	switch status {
	case "failed", "cancelled":
		return "error"
	default:
		return "info"
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
