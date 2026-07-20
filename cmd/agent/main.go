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
	case "apply-xray-config":
		return applyXrayConfig(task.Payload)
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

func applyXrayConfig(payload map[string]any) (string, error) {
	configText := payloadString(payload, "config", "")
	if strings.TrimSpace(configText) == "" {
		return "", errors.New("apply-xray-config payload.config is required")
	}
	if !json.Valid([]byte(configText)) {
		return "", errors.New("apply-xray-config payload.config must be valid JSON")
	}

	runtimeMode := payloadString(payload, "runtimeMode", "external")
	binaryPath := payloadString(payload, "binaryPath", "xray")
	configPath := payloadString(payload, "configPath", "/usr/local/etc/xray/config.json")
	serviceName := payloadString(payload, "serviceName", "xray")

	switch runtimeMode {
	case "external":
		return applyExternalXray(binaryPath, configPath, serviceName, configText)
	case "inline":
		return applyInlineXray(binaryPath, configPath, configText)
	default:
		return "", fmt.Errorf("unsupported xray runtime mode: %s", runtimeMode)
	}
}

func applyExternalXray(binaryPath string, configPath string, serviceName string, configText string) (string, error) {
	var output strings.Builder
	backupPath := configPath + ".harborx.bak"
	if existing, err := os.ReadFile(configPath); err == nil {
		if err := os.WriteFile(backupPath, existing, 0o600); err != nil {
			return output.String(), fmt.Errorf("backup current xray config: %w", err)
		}
		output.WriteString("backed up current config to " + backupPath + "\n")
	}
	if err := writeXrayConfigTo(configPath, configText); err != nil {
		return output.String(), err
	}
	output.WriteString("wrote xray config to " + configPath + "\n")

	testOutput, testErr := runCommand(60*time.Second, binaryPath, "test", "-config", configPath)
	output.WriteString(testOutput)
	if testErr != nil {
		rollbackOutput, rollbackErr := rollbackXrayConfig(configPath, backupPath, serviceName)
		output.WriteString(rollbackOutput)
		if rollbackErr != nil {
			return output.String(), fmt.Errorf("xray config test failed and rollback failed: %v; rollback: %w", testErr, rollbackErr)
		}
		return output.String(), fmt.Errorf("xray config test failed: %w", testErr)
	}

	restartOutput, restartErr := runCommand(60*time.Second, "systemctl", "restart", serviceName)
	output.WriteString(restartOutput)
	if restartErr != nil {
		rollbackOutput, rollbackErr := rollbackXrayConfig(configPath, backupPath, serviceName)
		output.WriteString(rollbackOutput)
		if rollbackErr != nil {
			return output.String(), fmt.Errorf("xray restart failed and rollback failed: %v; rollback: %w", restartErr, rollbackErr)
		}
		return output.String(), fmt.Errorf("xray restart failed: %w", restartErr)
	}
	output.WriteString("external xray restarted via systemd service " + serviceName + "\n")
	return output.String(), nil
}

func applyInlineXray(binaryPath string, configPath string, configText string) (string, error) {
	var output strings.Builder
	if err := writeXrayConfigTo(configPath, configText); err != nil {
		return output.String(), err
	}
	output.WriteString("wrote inline xray config to " + configPath + "\n")

	testOutput, testErr := runCommand(60*time.Second, binaryPath, "test", "-config", configPath)
	output.WriteString(testOutput)
	if testErr != nil {
		return output.String(), fmt.Errorf("inline xray config test failed: %w", testErr)
	}

	logPath := filepath.Join(filepath.Dir(configPath), "harborx-inline-xray.log")
	command := fmt.Sprintf("nohup %s run -config %s >> %s 2>&1 &", shellQuote(binaryPath), shellQuote(configPath), shellQuote(logPath))
	startOutput, startErr := runCommand(60*time.Second, "sh", "-lc", command)
	output.WriteString(startOutput)
	if startErr != nil {
		return output.String(), fmt.Errorf("start inline xray: %w", startErr)
	}
	output.WriteString("inline xray started; logs at " + logPath + "\n")
	return output.String(), nil
}

func rollbackXrayConfig(configPath string, backupPath string, serviceName string) (string, error) {
	if _, err := os.Stat(backupPath); err != nil {
		return "no previous xray config backup found\n", nil
	}
	backup, err := os.ReadFile(backupPath)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(configPath, backup, 0o600); err != nil {
		return "", err
	}
	output, err := runCommand(60*time.Second, "systemctl", "restart", serviceName)
	return "rolled back xray config\n" + output, err
}

func writeXrayConfig(configText string) error {
	if !json.Valid([]byte(configText)) {
		return errors.New("payload.config must be valid JSON")
	}
	configPath := "/usr/local/etc/xray/config.json"
	return writeXrayConfigTo(configPath, configText)
}

func writeXrayConfigTo(configPath string, configText string) error {
	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}
	return os.WriteFile(configPath, []byte(configText), 0o600)
}

func payloadString(payload map[string]any, key string, fallback string) string {
	if value, ok := payload[key].(string); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}
	return fallback
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\"'\"'") + "'"
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
