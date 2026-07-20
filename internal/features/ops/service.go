package ops

import (
	"errors"
	"strings"

	"harborx/internal/features/remote"
)

type Summary struct {
	ResourceKinds []string `json:"resourceKinds"`
	TaskKinds     []string `json:"taskKinds"`
	Capabilities  []string `json:"capabilities"`
}

type Resource struct {
	ID             string         `json:"id"`
	ResourceKind   string         `json:"resourceKind"`
	Name           string         `json:"name"`
	RemoteServerID string         `json:"remoteServerId"`
	Status         string         `json:"status"`
	Config         map[string]any `json:"config"`
	Enabled        bool           `json:"enabled"`
	CreatedAt      string         `json:"createdAt"`
	UpdatedAt      string         `json:"updatedAt"`
}

type CreateResourceInput struct {
	ResourceKind   string         `json:"resourceKind"`
	Name           string         `json:"name"`
	RemoteServerID string         `json:"remoteServerId"`
	Status         string         `json:"status"`
	Config         map[string]any `json:"config"`
	Enabled        bool           `json:"enabled"`
}

type ExecuteInput struct {
	Action string         `json:"action"`
	DryRun bool           `json:"dryRun"`
	Config map[string]any `json:"config"`
}

type ExecuteResult struct {
	Resource Resource       `json:"resource"`
	TaskID   string         `json:"taskId"`
	TaskKind string         `json:"taskKind"`
	Payload  map[string]any `json:"payload"`
	DryRun   bool           `json:"dryRun"`
}

type Repository interface {
	ListOpsResources(kind string) ([]Resource, error)
	CreateOpsResource(input CreateResourceInput) (Resource, error)
	UpdateOpsResource(id string, input CreateResourceInput) (Resource, error)
	DeleteOpsResource(id string) error
	GetOpsResource(id string) (Resource, error)
	CreateRemoteTask(serverID string, input remote.CreateTaskInput) (remote.RemoteTask, error)
	CreateRemoteTaskLog(serverID string, taskID string, eventKind string, message string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		ResourceKinds: supportedResourceKinds(),
		TaskKinds:     []string{"collect-xray-stats", "apply-nginx-config", "issue-certificate", "sync-external-subscription", "run-vps-maintenance", "apply-security-policy", "run-notification-automation"},
		Capabilities:  []string{"xray-inbounds", "traffic-stats", "acme", "nginx-fallback", "vps-ops", "subscription-sync", "security-audit", "notification-automation"},
	}
}

func (s Service) List(kind string) ([]Resource, error) {
	if s.repo == nil {
		return nil, errors.New("ops repository is not configured")
	}
	return s.repo.ListOpsResources(kind)
}

func (s Service) Create(input CreateResourceInput) (Resource, error) {
	if s.repo == nil {
		return Resource{}, errors.New("ops repository is not configured")
	}
	normalizeInput(&input)
	if err := validateInput(input); err != nil {
		return Resource{}, err
	}
	return s.repo.CreateOpsResource(input)
}

func (s Service) Update(id string, input CreateResourceInput) (Resource, error) {
	if s.repo == nil {
		return Resource{}, errors.New("ops repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return Resource{}, errors.New("ops resource id is required")
	}
	normalizeInput(&input)
	if err := validateInput(input); err != nil {
		return Resource{}, err
	}
	return s.repo.UpdateOpsResource(id, input)
}

func (s Service) Delete(id string) error {
	if s.repo == nil {
		return errors.New("ops repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("ops resource id is required")
	}
	return s.repo.DeleteOpsResource(id)
}

func (s Service) Execute(id string, input ExecuteInput) (ExecuteResult, error) {
	if s.repo == nil {
		return ExecuteResult{}, errors.New("ops repository is not configured")
	}
	resource, err := s.repo.GetOpsResource(id)
	if err != nil {
		return ExecuteResult{}, err
	}
	if !resource.Enabled {
		return ExecuteResult{}, errors.New("ops resource is disabled")
	}
	action := strings.TrimSpace(input.Action)
	if action == "" {
		action = defaultActionForKind(resource.ResourceKind)
	}
	taskKind := taskKindFor(resource.ResourceKind, action)
	if taskKind == "" {
		return ExecuteResult{}, errors.New("unsupported ops action for resource kind")
	}
	payload := mergeConfig(resource.Config, input.Config)
	payload["resourceId"] = resource.ID
	payload["resourceKind"] = resource.ResourceKind
	payload["resourceName"] = resource.Name
	payload["action"] = action
	result := ExecuteResult{
		Resource: resource,
		TaskKind: taskKind,
		Payload:  payload,
		DryRun:   input.DryRun,
	}
	if input.DryRun {
		return result, nil
	}
	if strings.TrimSpace(resource.RemoteServerID) == "" {
		return ExecuteResult{}, errors.New("ops resource must be bound to a remote server before execute")
	}
	task, err := s.repo.CreateRemoteTask(resource.RemoteServerID, remote.CreateTaskInput{
		TaskKind: taskKind,
		Payload:  payload,
	})
	if err != nil {
		return ExecuteResult{}, err
	}
	_ = s.repo.CreateRemoteTaskLog(resource.RemoteServerID, task.ID, "queued", "ops task queued: "+taskKind)
	result.TaskID = task.ID
	return result, nil
}

func normalizeInput(input *CreateResourceInput) {
	input.ResourceKind = strings.TrimSpace(input.ResourceKind)
	input.Name = strings.TrimSpace(input.Name)
	input.RemoteServerID = strings.TrimSpace(input.RemoteServerID)
	input.Status = strings.TrimSpace(input.Status)
	if input.Status == "" {
		input.Status = "active"
	}
}

func validateInput(input CreateResourceInput) error {
	if input.Name == "" {
		return errors.New("ops resource name is required")
	}
	if !supportedResourceKind(input.ResourceKind) {
		return errors.New("unsupported ops resource kind")
	}
	return nil
}

func supportedResourceKind(kind string) bool {
	for _, item := range supportedResourceKinds() {
		if kind == item {
			return true
		}
	}
	return false
}

func supportedResourceKinds() []string {
	return []string{"xray-inbound", "traffic-collector", "certificate-automation", "nginx-fallback", "vps-maintenance", "external-subscription", "security-policy", "notification-automation"}
}

func defaultActionForKind(kind string) string {
	switch kind {
	case "xray-inbound":
		return "render"
	case "traffic-collector":
		return "collect"
	case "certificate-automation":
		return "issue"
	case "nginx-fallback":
		return "apply"
	case "vps-maintenance":
		return "run"
	case "external-subscription":
		return "sync"
	case "security-policy":
		return "apply"
	case "notification-automation":
		return "run"
	default:
		return ""
	}
}

func taskKindFor(kind string, action string) string {
	switch kind {
	case "traffic-collector":
		return "collect-xray-stats"
	case "certificate-automation":
		return "issue-certificate"
	case "nginx-fallback":
		return "apply-nginx-config"
	case "vps-maintenance":
		return "run-vps-maintenance"
	case "external-subscription":
		return "sync-external-subscription"
	case "security-policy":
		return "apply-security-policy"
	case "notification-automation":
		return "run-notification-automation"
	case "xray-inbound":
		if action == "apply" {
			return "apply-xray-config"
		}
		return "render-xray-inbound"
	default:
		return ""
	}
}

func mergeConfig(base map[string]any, overlay map[string]any) map[string]any {
	output := map[string]any{}
	for key, value := range base {
		output[key] = value
	}
	for key, value := range overlay {
		output[key] = value
	}
	return output
}
