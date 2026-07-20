package subscriptions

import (
	"bytes"
	"errors"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"harborx/internal/features/nodes"
	"harborx/internal/features/rules"
	"harborx/internal/features/templates"
)

type Subscription struct {
	ID           string         `json:"id"`
	Name         string         `json:"name"`
	OwnerUserID  string         `json:"ownerUserId"`
	OutputFormat string         `json:"outputFormat"`
	TemplateID   string         `json:"templateId"`
	Sources      []string       `json:"sources"`
	Options      map[string]any `json:"options"`
	CreatedAt    string         `json:"createdAt"`
	UpdatedAt    string         `json:"updatedAt"`
}

type CreateInput struct {
	Name         string         `json:"name"`
	OwnerUserID  string         `json:"ownerUserId"`
	OutputFormat string         `json:"outputFormat"`
	TemplateID   string         `json:"templateId"`
	Sources      []string       `json:"sources"`
	Options      map[string]any `json:"options"`
}

type Summary struct {
	OutputFormats []string `json:"outputFormats"`
	Capabilities  []string `json:"capabilities"`
}

type Repository interface {
	ListSubscriptions() ([]Subscription, error)
	CreateSubscription(input CreateInput) (Subscription, error)
	ListNodes() ([]nodes.Node, error)
	ListRuleSets() ([]rules.RuleSet, error)
	ListTemplates() ([]templates.Template, error)
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Summary() Summary {
	return Summary{
		OutputFormats: []string{
			"clash-meta",
			"surge",
			"loon",
			"quantumult-x",
			"shadowrocket",
			"sing-box",
			"stash",
			"surfboard",
			"v2ray",
		},
		Capabilities: []string{"per-user-subscribe", "template-render", "short-links", "merge-sources"},
	}
}

func (s Service) List() ([]Subscription, error) {
	if s.repo == nil {
		return nil, errors.New("subscriptions repository is not configured")
	}
	return s.repo.ListSubscriptions()
}

func (s Service) Create(input CreateInput) (Subscription, error) {
	if s.repo == nil {
		return Subscription{}, errors.New("subscriptions repository is not configured")
	}
	return s.repo.CreateSubscription(input)
}

type RenderedSubscription struct {
	SubscriptionID string `json:"subscriptionId"`
	Name           string `json:"name"`
	OutputFormat   string `json:"outputFormat"`
	TemplateID     string `json:"templateId"`
	Content        string `json:"content"`
	FileName       string `json:"fileName"`
	ContentType    string `json:"contentType"`
}

type RenderContext struct {
	SubscriptionName string
	OutputFormat     string
	Nodes            []nodes.Node
	RuleSets         []rules.RuleSet
	Rules            string
	Proxies          string
	ProxyGroups      string
	Outbounds        string
	RouteRules       string
	DNS              string
}

func (s Service) Render(id string) (RenderedSubscription, error) {
	if s.repo == nil {
		return RenderedSubscription{}, errors.New("subscriptions repository is not configured")
	}

	subscription, err := s.findSubscription(id)
	if err != nil {
		return RenderedSubscription{}, err
	}

	templateRecord, err := s.findTemplate(subscription.TemplateID)
	if err != nil {
		return RenderedSubscription{}, err
	}

	nodeItems, err := s.repo.ListNodes()
	if err != nil {
		return RenderedSubscription{}, err
	}

	ruleSets, err := s.repo.ListRuleSets()
	if err != nil {
		return RenderedSubscription{}, err
	}

	context := RenderContext{
		SubscriptionName: subscription.Name,
		OutputFormat:     subscription.OutputFormat,
		Nodes:            nodeItems,
		RuleSets:         ruleSets,
		Rules:            renderClashRules(ruleSets),
		Proxies:          renderClashProxies(nodeItems),
		ProxyGroups:      renderClashProxyGroups(nodeItems),
		Outbounds:        renderSingBoxOutbounds(nodeItems),
		RouteRules:       renderSingBoxRouteRules(ruleSets),
		DNS:              renderDefaultDNS(),
	}

	parsedTemplate, err := template.New(templateRecord.ID).Parse(templateRecord.Content)
	if err != nil {
		return RenderedSubscription{}, fmt.Errorf("parse template: %w", err)
	}

	var output bytes.Buffer
	if err := parsedTemplate.Execute(&output, context); err != nil {
		return RenderedSubscription{}, fmt.Errorf("render template: %w", err)
	}

	return RenderedSubscription{
		SubscriptionID: subscription.ID,
		Name:           subscription.Name,
		OutputFormat:   subscription.OutputFormat,
		TemplateID:     subscription.TemplateID,
		Content:        strings.TrimSpace(output.String()) + "\n",
		FileName:       subscriptionFileName(subscription),
		ContentType:    contentTypeForFormat(subscription.OutputFormat),
	}, nil
}

func (s Service) findSubscription(id string) (Subscription, error) {
	items, err := s.repo.ListSubscriptions()
	if err != nil {
		return Subscription{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return Subscription{}, errors.New("subscription not found")
}

func (s Service) findTemplate(id string) (templates.Template, error) {
	items, err := s.repo.ListTemplates()
	if err != nil {
		return templates.Template{}, err
	}
	for _, item := range items {
		if item.ID == id {
			return item, nil
		}
	}
	return templates.Template{}, errors.New("template not found")
}

func renderClashProxies(items []nodes.Node) string {
	if len(items) == 0 {
		return "  []"
	}

	lines := make([]string, 0, len(items))
	for _, item := range sortedNodes(items) {
		if !item.Enabled {
			continue
		}
		lines = append(lines, fmt.Sprintf("  - name: %q\n    type: %s\n    server: %s\n    port: %d", item.Name, clashProtocol(item.Protocol), item.ServerHost, item.ServerPort))
	}
	if len(lines) == 0 {
		return "  []"
	}
	return strings.Join(lines, "\n")
}

func renderClashProxyGroups(items []nodes.Node) string {
	names := enabledNodeNames(items)
	if len(names) == 0 {
		return "  - name: Proxy\n    type: select\n    proxies:\n      - DIRECT"
	}

	lines := []string{"  - name: Proxy", "    type: select", "    proxies:"}
	for _, name := range names {
		lines = append(lines, fmt.Sprintf("      - %q", name))
	}
	lines = append(lines, "      - DIRECT")
	return strings.Join(lines, "\n")
}

func renderClashRules(ruleSets []rules.RuleSet) string {
	var lines []string
	for _, set := range ruleSets {
		ruleItems := append([]rules.Rule(nil), set.Rules...)
		sort.SliceStable(ruleItems, func(i, j int) bool {
			return ruleItems[i].SortOrder < ruleItems[j].SortOrder
		})
		for _, rule := range ruleItems {
			if !rule.Enabled {
				continue
			}
			if rule.RuleType == "MATCH" {
				lines = append(lines, fmt.Sprintf("  - MATCH,%s", rule.Policy))
				continue
			}
			lines = append(lines, fmt.Sprintf("  - %s,%s,%s", rule.RuleType, rule.Pattern, rule.Policy))
		}
	}
	if len(lines) == 0 {
		return "  - MATCH,Proxy"
	}
	return strings.Join(lines, "\n")
}

func renderSingBoxOutbounds(items []nodes.Node) string {
	var lines []string
	for _, item := range sortedNodes(items) {
		if !item.Enabled {
			continue
		}
		lines = append(lines, fmt.Sprintf(`{"type":"%s","tag":%q,"server":%q,"server_port":%d}`, singBoxProtocol(item.Protocol), item.Name, item.ServerHost, item.ServerPort))
	}
	if len(lines) == 0 {
		return `[{"type":"direct","tag":"DIRECT"}]`
	}
	lines = append([]string{`{"type":"selector","tag":"Proxy","outbounds":[` + quotedList(enabledNodeNames(items)) + `,"DIRECT"]}`}, lines...)
	lines = append(lines, `{"type":"direct","tag":"DIRECT"}`)
	return "[" + strings.Join(lines, ",") + "]"
}

func renderSingBoxRouteRules(ruleSets []rules.RuleSet) string {
	var lines []string
	for _, set := range ruleSets {
		for _, rule := range set.Rules {
			if !rule.Enabled || rule.RuleType == "MATCH" {
				continue
			}
			field := singBoxRuleField(rule.RuleType)
			if field == "" {
				continue
			}
			lines = append(lines, fmt.Sprintf(`{"%s":[%q],"outbound":%q}`, field, rule.Pattern, rule.Policy))
		}
	}
	if len(lines) == 0 {
		return "[]"
	}
	return "[" + strings.Join(lines, ",") + "]"
}

func renderDefaultDNS() string {
	return "  enable: true\n  listen: 0.0.0.0:1053\n  nameserver:\n    - 223.5.5.5\n    - 8.8.8.8"
}

func sortedNodes(items []nodes.Node) []nodes.Node {
	copied := append([]nodes.Node(nil), items...)
	sort.SliceStable(copied, func(i, j int) bool {
		return copied[i].Name < copied[j].Name
	})
	return copied
}

func enabledNodeNames(items []nodes.Node) []string {
	var names []string
	for _, item := range sortedNodes(items) {
		if item.Enabled {
			names = append(names, item.Name)
		}
	}
	return names
}

func quotedList(items []string) string {
	quoted := make([]string, 0, len(items))
	for _, item := range items {
		quoted = append(quoted, fmt.Sprintf("%q", item))
	}
	return strings.Join(quoted, ",")
}

func clashProtocol(protocol string) string {
	switch strings.ToLower(protocol) {
	case "shadowsocks":
		return "ss"
	default:
		return strings.ToLower(protocol)
	}
}

func singBoxProtocol(protocol string) string {
	switch strings.ToLower(protocol) {
	case "shadowsocks":
		return "shadowsocks"
	case "socks5":
		return "socks"
	default:
		return strings.ToLower(protocol)
	}
}

func singBoxRuleField(ruleType string) string {
	switch ruleType {
	case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD":
		return "domain"
	case "IP-CIDR", "SRC-IP-CIDR":
		return "ip_cidr"
	case "GEOIP":
		return "geoip"
	default:
		return ""
	}
}

func subscriptionFileName(subscription Subscription) string {
	extension := "txt"
	switch subscription.OutputFormat {
	case "clash-meta", "surge", "loon", "stash", "surfboard":
		extension = "yaml"
	case "sing-box":
		extension = "json"
	}
	name := strings.ToLower(subscription.Name)
	name = strings.NewReplacer(" ", "-", "/", "-", "\\", "-", ":", "-").Replace(name)
	if name == "" {
		name = "subscription"
	}
	return name + "." + extension
}

func contentTypeForFormat(format string) string {
	switch format {
	case "sing-box":
		return "application/json; charset=utf-8"
	default:
		return "text/yaml; charset=utf-8"
	}
}
