package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type remoteServer struct {
	ID     string `json:"id"`
	Name   string `json:"name"`
	Host   string `json:"host"`
	Status string `json:"status"`
}

type remoteTask struct {
	ID       string         `json:"id"`
	TaskKind string         `json:"taskKind"`
	Status   string         `json:"status"`
	Payload  map[string]any `json:"payload"`
}

type taskClaim struct {
	Server remoteServer `json:"server"`
	Task   *remoteTask  `json:"task"`
}

type agentConfig struct {
	BaseURL    string
	Token      string
	Interval   time.Duration
	AllowShell bool
}

func main() {
	cfg, err := loadConfig()
	if err != nil {
		log.Fatal(err)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	log.Printf("harborx agent starting for %s", cfg.BaseURL)

	for {
		if err := heartbeat(client, cfg); err != nil {
			log.Printf("heartbeat failed: %v", err)
		}
		if err := claimAndRun(client, cfg); err != nil {
			log.Printf("task loop failed: %v", err)
		}
		time.Sleep(cfg.Interval)
	}
}

func loadConfig() (agentConfig, error) {
	cfg := agentConfig{
		BaseURL:    strings.TrimRight(os.Getenv("HARBORX_AGENT_BASE_URL"), "/"),
		Token:      os.Getenv("HARBORX_AGENT_TOKEN"),
		Interval:   10 * time.Second,
		AllowShell: os.Getenv("HARBORX_AGENT_ALLOW_SHELL") == "1",
	}
	if cfg.BaseURL == "" {
		return agentConfig{}, errors.New("HARBORX_AGENT_BASE_URL is required")
	}
	if cfg.Token == "" {
		return agentConfig{}, errors.New("HARBORX_AGENT_TOKEN is required")
	}
	if raw := os.Getenv("HARBORX_AGENT_INTERVAL_SECONDS"); raw != "" {
		seconds, err := strconv.Atoi(raw)
		if err != nil || seconds <= 0 {
			return agentConfig{}, errors.New("HARBORX_AGENT_INTERVAL_SECONDS must be a positive integer")
		}
		cfg.Interval = time.Duration(seconds) * time.Second
	}
	return cfg, nil
}

func heartbeat(client *http.Client, cfg agentConfig) error {
	hostname, _ := os.Hostname()
	payload := map[string]any{
		"status": "online",
		"metadata": map[string]any{
			"hostname": hostname,
			"os":       runtime.GOOS,
			"arch":     runtime.GOARCH,
			"agent":    "harborx-agent",
		},
	}
	return postJSON(client, cfg, "/api/v1/agent/heartbeat", payload, nil)
}

func claimAndRun(client *http.Client, cfg agentConfig) error {
	var claim taskClaim
	if err := postJSON(client, cfg, "/api/v1/agent/tasks/claim", map[string]any{}, &claim); err != nil {
		return err
	}
	if claim.Task == nil {
		return nil
	}

	task := *claim.Task
	log.Printf("claimed task %s (%s)", task.ID, task.TaskKind)
	_ = sendAgentLog(client, cfg, "info", "claimed task "+task.ID+" ("+task.TaskKind+")", map[string]any{
		"taskId":   task.ID,
		"taskKind": task.TaskKind,
	})

	output, err := runTask(cfg, task)
	status := "succeeded"
	if err != nil {
		status = "failed"
		if output == "" {
			output = err.Error()
		} else {
			output += "\n" + err.Error()
		}
	}
	logLevel := "info"
	if status == "failed" {
		logLevel = "error"
	}
	_ = sendAgentLog(client, cfg, logLevel, "task "+task.ID+" "+status, map[string]any{
		"taskId":   task.ID,
		"taskKind": task.TaskKind,
		"output":   output,
	})

	update := map[string]any{
		"status":     status,
		"outputText": output,
	}
	return postJSON(client, cfg, "/api/v1/agent/tasks/"+task.ID, update, nil)
}

func sendAgentLog(client *http.Client, cfg agentConfig, level string, message string, metadata map[string]any) error {
	payload := map[string]any{
		"level":    level,
		"message":  message,
		"metadata": metadata,
	}
	return postJSON(client, cfg, "/api/v1/agent/logs", payload, nil)
}

func runTask(cfg agentConfig, task remoteTask) (string, error) {
	switch task.TaskKind {
	case "restart-xray":
		return runCommand(60*time.Second, "systemctl", "restart", "xray")
	case "reload-config":
		if configText, _ := task.Payload["config"].(string); strings.TrimSpace(configText) != "" {
			if err := writeXrayConfig(configText); err != nil {
				return "", err
			}
			return runCommand(60*time.Second, "systemctl", "restart", "xray")
		}
		return runCommand(60*time.Second, "systemctl", "reload", "xray")
	case "install-nginx":
		return runCommand(5*time.Minute, "sh", "-lc", packageInstallCommand("nginx"))
	case "install-warp":
		return runCommand(10*time.Minute, "sh", "-lc", "curl -fsSL https://raw.githubusercontent.com/fscarmen/warp/main/menu.sh -o /tmp/warp-menu.sh && bash /tmp/warp-menu.sh")
	case "install-xray":
		return runCommand(10*time.Minute, "sh", "-lc", "bash -c \"$(curl -L https://github.com/XTLS/Xray-install/raw/main/install-release.sh)\" @ install")
	case "renew-certificate":
		return runCommand(5*time.Minute, "sh", "-lc", "command -v certbot >/dev/null && certbot renew --quiet")
	case "shell-script":
		if !cfg.AllowShell {
			return "", errors.New("shell-script is disabled; set HARBORX_AGENT_ALLOW_SHELL=1 to enable")
		}
		command, _ := task.Payload["command"].(string)
		if strings.TrimSpace(command) == "" {
			return "", errors.New("shell-script payload.command is required")
		}
		return runCommand(10*time.Minute, "sh", "-lc", command)
	default:
		return "", fmt.Errorf("unsupported task kind: %s", task.TaskKind)
	}
}

func writeXrayConfig(configText string) error {
	if !json.Valid([]byte(configText)) {
		return errors.New("payload.config must be valid JSON")
	}
	configPath := "/usr/local/etc/xray/config.json"
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(configText), 0o600)
}

func packageInstallCommand(packageName string) string {
	return fmt.Sprintf(`if command -v %s >/dev/null 2>&1; then exit 0; fi
if command -v apt-get >/dev/null 2>&1; then apt-get update && apt-get install -y %s; exit $?; fi
if command -v dnf >/dev/null 2>&1; then dnf install -y %s; exit $?; fi
if command -v yum >/dev/null 2>&1; then yum install -y %s; exit $?; fi
echo "unsupported package manager"; exit 1`, packageName, packageName, packageName, packageName)
}

func runCommand(timeout time.Duration, name string, args ...string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	if ctx.Err() == context.DeadlineExceeded {
		return string(output), errors.New("command timed out")
	}
	return string(output), err
}

func postJSON(client *http.Client, cfg agentConfig, path string, payload any, target any) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	request, err := http.NewRequest(http.MethodPost, cfg.BaseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	request.Header.Set("Content-Type", "application/json")
	request.Header.Set("X-HarborX-Agent-Token", cfg.Token)

	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()

	if response.StatusCode < 200 || response.StatusCode >= 300 {
		var apiError struct {
			Error string `json:"error"`
		}
		_ = json.NewDecoder(response.Body).Decode(&apiError)
		if apiError.Error != "" {
			return errors.New(apiError.Error)
		}
		return fmt.Errorf("request failed: %s", response.Status)
	}
	if target == nil {
		return nil
	}
	return json.NewDecoder(response.Body).Decode(target)
}
