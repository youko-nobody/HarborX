package nodes

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net"
	"net/url"
	"strconv"
	"strings"
)

func ParseShareLinks(input ImportInput) ([]CreateInput, []string) {
	sourceKind := strings.TrimSpace(input.SourceKind)
	if sourceKind == "" {
		sourceKind = "share-link-import"
	}

	var created []CreateInput
	var skipped []string
	for _, token := range extractShareTokens(input.Content) {
		item, err := parseShareToken(token, sourceKind, input.Tags)
		if err != nil {
			skipped = append(skipped, token+": "+err.Error())
			continue
		}
		created = append(created, item)
	}
	return created, skipped
}

func extractShareTokens(content string) []string {
	content = strings.NewReplacer("\r", "\n", "\t", "\n", " ", "\n").Replace(content)
	var tokens []string
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "vmess://") || strings.HasPrefix(line, "vless://") || strings.HasPrefix(line, "trojan://") || strings.HasPrefix(line, "ss://") {
			tokens = append(tokens, line)
		}
	}
	return tokens
}

func parseShareToken(token string, sourceKind string, tags []string) (CreateInput, error) {
	switch {
	case strings.HasPrefix(token, "vmess://"):
		return parseVMess(token, sourceKind, tags)
	case strings.HasPrefix(token, "vless://"):
		return parseURLShare(token, "vless", sourceKind, tags)
	case strings.HasPrefix(token, "trojan://"):
		return parseURLShare(token, "trojan", sourceKind, tags)
	case strings.HasPrefix(token, "ss://"):
		return parseShadowsocks(token, sourceKind, tags)
	default:
		return CreateInput{}, fmt.Errorf("unsupported share link")
	}
}

func parseVMess(token string, sourceKind string, tags []string) (CreateInput, error) {
	raw := strings.TrimPrefix(token, "vmess://")
	data, err := decodeBase64(raw)
	if err != nil {
		return CreateInput{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return CreateInput{}, err
	}
	host := stringFromMap(payload, "add")
	port, _ := strconv.Atoi(stringFromMap(payload, "port"))
	name := stringFromMap(payload, "ps")
	if name == "" {
		name = host
	}
	return createImportedNode(name, sourceKind, "vmess", host, port, tags, payload)
}

func parseURLShare(token string, protocol string, sourceKind string, tags []string) (CreateInput, error) {
	parsed, err := url.Parse(token)
	if err != nil {
		return CreateInput{}, err
	}
	host := parsed.Hostname()
	port, _ := strconv.Atoi(parsed.Port())
	name := strings.TrimSpace(parsed.Fragment)
	if name == "" {
		name = host
	}
	metadata := map[string]any{
		"user":   parsed.User.Username(),
		"query":  parsed.Query(),
		"rawURL": token,
	}
	return createImportedNode(name, sourceKind, protocol, host, port, tags, metadata)
}

func parseShadowsocks(token string, sourceKind string, tags []string) (CreateInput, error) {
	parsed, err := url.Parse(token)
	if err != nil {
		return CreateInput{}, err
	}
	name := strings.TrimSpace(parsed.Fragment)
	host := parsed.Hostname()
	port, _ := strconv.Atoi(parsed.Port())
	methodAndPassword := parsed.User.String()
	if host == "" {
		withoutScheme := strings.TrimPrefix(token, "ss://")
		mainPart := strings.SplitN(withoutScheme, "#", 2)[0]
		decoded, err := decodeBase64(mainPart)
		if err == nil {
			methodAndHost := string(decoded)
			if at := strings.LastIndex(methodAndHost, "@"); at >= 0 {
				methodAndPassword = methodAndHost[:at]
				hostPart := methodAndHost[at+1:]
				splitHost, splitPort, _ := net.SplitHostPort(hostPart)
				portInt, _ := strconv.Atoi(splitPort)
				if portInt > 0 {
					host = splitHost
					metadata := map[string]any{"methodAndPassword": methodAndPassword, "rawURL": token}
					if name == "" {
						name = host
					}
					return createImportedNode(name, sourceKind, "shadowsocks", host, portInt, tags, metadata)
				}
			}
		}
	}
	if name == "" {
		name = host
	}
	return createImportedNode(name, sourceKind, "shadowsocks", host, port, tags, map[string]any{
		"methodAndPassword": methodAndPassword,
		"query":             parsed.Query(),
		"rawURL":            token,
	})
}

func createImportedNode(name string, sourceKind string, protocol string, host string, port int, tags []string, metadata map[string]any) (CreateInput, error) {
	if strings.TrimSpace(host) == "" {
		return CreateInput{}, fmt.Errorf("host is required")
	}
	if port <= 0 {
		return CreateInput{}, fmt.Errorf("port is required")
	}
	return CreateInput{
		Name:       name,
		SourceKind: sourceKind,
		Protocol:   protocol,
		ServerHost: host,
		ServerPort: port,
		Tags:       tags,
		Metadata:   metadata,
		Enabled:    true,
	}, nil
}

func decodeBase64(raw string) ([]byte, error) {
	raw = strings.TrimSpace(raw)
	if data, err := base64.RawURLEncoding.DecodeString(raw); err == nil {
		return data, nil
	}
	if data, err := base64.RawStdEncoding.DecodeString(raw); err == nil {
		return data, nil
	}
	if data, err := base64.StdEncoding.DecodeString(raw); err == nil {
		return data, nil
	}
	return nil, fmt.Errorf("invalid base64 payload")
}

func stringFromMap(input map[string]any, key string) string {
	value, _ := input[key].(string)
	return strings.TrimSpace(value)
}
