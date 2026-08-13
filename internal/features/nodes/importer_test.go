package nodes

import (
	"encoding/base64"
	"encoding/json"
	"net/url"
	"strings"
	"testing"
)

func TestExtractShareTokens(t *testing.T) {
	content := "some junk\nvmess://abc\nvless://def\ntrojan://ghi\nss://jkl\nhttp://not-a-token\n"
	tokens := extractShareTokens(content)
	if len(tokens) != 4 {
		t.Fatalf("expected 4 tokens, got %d: %v", len(tokens), tokens)
	}
	for _, tok := range tokens {
		if !strings.HasPrefix(tok, "vmess://") && !strings.HasPrefix(tok, "vless://") &&
			!strings.HasPrefix(tok, "trojan://") && !strings.HasPrefix(tok, "ss://") {
			t.Fatalf("unexpected token: %q", tok)
		}
	}
}

func TestParseVmess(t *testing.T) {
	payload := map[string]any{
		"add":  "example.com",
		"port": "443",
		"id":   "uuid-here",
		"aid":  "0",
		"net":  "ws",
		"ps":   "Tokyo VMess",
	}
	raw, _ := json.Marshal(payload)
	link := "vmess://" + base64.RawURLEncoding.EncodeToString(raw)

	items, skipped := ParseShareLinks(ImportInput{Content: link})
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Protocol != "vmess" {
		t.Fatalf("unexpected protocol: %q", item.Protocol)
	}
	if item.ServerHost != "example.com" {
		t.Fatalf("unexpected host: %q", item.ServerHost)
	}
	if item.ServerPort != 443 {
		t.Fatalf("unexpected port: %d", item.ServerPort)
	}
	if item.Name != "Tokyo VMess" {
		t.Fatalf("unexpected name: %q", item.Name)
	}
	if item.Metadata["id"] != "uuid-here" {
		t.Fatalf("metadata id not preserved: %v", item.Metadata)
	}
}

func TestParseVless(t *testing.T) {
	link := "vless://uuid@example.com:8443?encryption=none&security=reality&flow=xtls-rprx-vision#Tokyo%20VLESS"
	items, skipped := ParseShareLinks(ImportInput{Content: link})
	if len(skipped) != 0 {
		t.Fatalf("unexpected skipped: %v", skipped)
	}
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	item := items[0]
	if item.Protocol != "vless" {
		t.Fatalf("unexpected protocol: %q", item.Protocol)
	}
	if item.ServerHost != "example.com" || item.ServerPort != 8443 {
		t.Fatalf("unexpected host/port: %s:%d", item.ServerHost, item.ServerPort)
	}
	if item.Name != "Tokyo VLESS" {
		t.Fatalf("unexpected name: %q", item.Name)
	}
	query, ok := item.Metadata["query"].(url.Values)
	if !ok || query.Get("security") != "reality" {
		t.Fatalf("query metadata not preserved: %v", item.Metadata["query"])
	}
}

func TestParseTrojan(t *testing.T) {
	link := "trojan://password@trojan.example.com:443?security=tls&sni=example.com#Trojan%20Node"
	items, skipped := ParseShareLinks(ImportInput{Content: link})
	if len(skipped) != 0 || len(items) != 1 {
		t.Fatalf("unexpected result: items=%d skipped=%v", len(items), skipped)
	}
	item := items[0]
	if item.Protocol != "trojan" || item.ServerHost != "trojan.example.com" || item.ServerPort != 443 {
		t.Fatalf("unexpected parse: %+v", item)
	}
	if item.Name != "Trojan Node" {
		t.Fatalf("unexpected name: %q", item.Name)
	}
}

func TestParseShadowsocksURLFormat(t *testing.T) {
	// ss://base64(method:password)@host:port#name
	credential := base64.RawURLEncoding.EncodeToString([]byte("aes-256-gcm:password"))
	link := "ss://" + credential + "@ss.example.com:8388#SS%20Node"
	items, skipped := ParseShareLinks(ImportInput{Content: link})
	if len(skipped) != 0 || len(items) != 1 {
		t.Fatalf("unexpected result: items=%d skipped=%v", len(items), skipped)
	}
	item := items[0]
	if item.Protocol != "shadowsocks" || item.ServerHost != "ss.example.com" || item.ServerPort != 8388 {
		t.Fatalf("unexpected parse: %+v", item)
	}
	if item.Name != "SS Node" {
		t.Fatalf("unexpected name: %q", item.Name)
	}
	// The base64 credential must be decoded into plain method:password.
	methodAndPassword, ok := item.Metadata["methodAndPassword"].(string)
	if !ok || methodAndPassword != "aes-256-gcm:password" {
		t.Fatalf("credential not decoded: %q", methodAndPassword)
	}
}

func TestParseShadowsocksLegacyFormat(t *testing.T) {
	// Legacy ss://base64(method:password@host:port) without URI fragment.
	plain := "aes-256-gcm:password@legacy.example.com:10086"
	link := "ss://" + base64.RawURLEncoding.EncodeToString([]byte(plain))
	items, skipped := ParseShareLinks(ImportInput{Content: link})
	if len(skipped) != 0 || len(items) != 1 {
		t.Fatalf("unexpected result: items=%d skipped=%v", len(items), skipped)
	}
	item := items[0]
	if item.Protocol != "shadowsocks" || item.ServerHost != "legacy.example.com" || item.ServerPort != 10086 {
		t.Fatalf("unexpected parse: %+v", item)
	}
	methodAndPassword, ok := item.Metadata["methodAndPassword"].(string)
	if !ok || methodAndPassword != "aes-256-gcm:password" {
		t.Fatalf("credential not decoded: %q", methodAndPassword)
	}
}

func TestParseSkipsJunk(t *testing.T) {
	content := "not a link\nvmess://%%%invalid%%%\n"
	items, skipped := ParseShareLinks(ImportInput{Content: content})
	if len(items) != 0 {
		t.Fatalf("expected 0 items, got %d", len(items))
	}
	if len(skipped) != 1 {
		t.Fatalf("expected 1 skipped, got %d: %v", len(skipped), skipped)
	}
}

func TestParseShareLinksEmpty(t *testing.T) {
	items, skipped := ParseShareLinks(ImportInput{Content: ""})
	if len(items) != 0 || len(skipped) != 0 {
		t.Fatalf("expected empty result, got items=%d skipped=%v", len(items), skipped)
	}
}

func TestParseShareLinksAppliesTags(t *testing.T) {
	link := "vless://uuid@example.com:443#Node"
	items, _ := ParseShareLinks(ImportInput{Content: link, Tags: []string{"hk", "xray"}})
	if len(items) != 1 {
		t.Fatalf("expected 1 item, got %d", len(items))
	}
	found := false
	for _, tag := range items[0].Tags {
		if tag == "hk" || tag == "xray" {
			found = true
		}
	}
	if !found {
		t.Fatalf("tags not applied: %v", items[0].Tags)
	}
}
