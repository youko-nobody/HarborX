package rules

import (
	"errors"
	"fmt"
	"net"
	"strings"
)

type RuleType struct {
	Key           string `json:"key"`
	Label         string `json:"label"`
	PatternHint   string `json:"patternHint"`
	Example       string `json:"example"`
	SupportsNoArg bool   `json:"supportsNoArg"`
}

type Rule struct {
	ID        string `json:"id"`
	RuleType  string `json:"ruleType"`
	Pattern   string `json:"pattern"`
	Policy    string `json:"policy"`
	SortOrder int    `json:"sortOrder"`
	Enabled   bool   `json:"enabled"`
	Note      string `json:"note"`
}

type RuleSet struct {
	ID          string `json:"id"`
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	CreatedAt   string `json:"createdAt"`
	UpdatedAt   string `json:"updatedAt"`
	Rules       []Rule `json:"rules"`
}

type CreateRuleSetInput struct {
	Name        string `json:"name"`
	Scope       string `json:"scope"`
	Description string `json:"description"`
	Rules       []Rule `json:"rules"`
}

type ValidationIssue struct {
	Path    string `json:"path"`
	Message string `json:"message"`
}

type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Issues []ValidationIssue `json:"issues"`
}

type Bootstrap struct {
	RuleTypes      []RuleType `json:"ruleTypes"`
	Policies       []string   `json:"policies"`
	DefaultRules   []string   `json:"defaultRules"`
	TemplateIDs    []string   `json:"templateIds"`
	EditorFeatures []string   `json:"editorFeatures"`
}

type Repository interface {
	ListRuleSets() ([]RuleSet, error)
	CreateRuleSet(input CreateRuleSetInput) (RuleSet, error)
	UpdateRuleSet(id string, input CreateRuleSetInput) (RuleSet, error)
	DeleteRuleSet(id string) error
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) Service {
	return Service{repo: repo}
}

func (Service) Bootstrap() Bootstrap {
	return Bootstrap{
		RuleTypes: []RuleType{
			{Key: "DOMAIN", Label: "Domain", PatternHint: "example.com", Example: "DOMAIN,example.com,DIRECT"},
			{Key: "DOMAIN-SUFFIX", Label: "Domain Suffix", PatternHint: "google.com", Example: "DOMAIN-SUFFIX,google.com,Proxy"},
			{Key: "DOMAIN-KEYWORD", Label: "Domain Keyword", PatternHint: "telegram", Example: "DOMAIN-KEYWORD,telegram,Proxy"},
			{Key: "IP-CIDR", Label: "IP CIDR", PatternHint: "1.1.1.0/24", Example: "IP-CIDR,1.1.1.0/24,DIRECT"},
			{Key: "SRC-IP-CIDR", Label: "Source IP CIDR", PatternHint: "192.168.1.0/24", Example: "SRC-IP-CIDR,192.168.1.0/24,DIRECT"},
			{Key: "GEOIP", Label: "GeoIP", PatternHint: "CN", Example: "GEOIP,CN,DIRECT"},
			{Key: "MATCH", Label: "Match", PatternHint: "No value needed", Example: "MATCH,Proxy", SupportsNoArg: true},
		},
		Policies: []string{
			"DIRECT",
			"REJECT",
			"Proxy",
			"Auto",
			"Fallback",
			"Domestic",
			"Global",
		},
		DefaultRules: []string{
			"DOMAIN-SUFFIX,local,DIRECT",
			"DOMAIN-SUFFIX,lan,DIRECT",
			"GEOIP,CN,DIRECT",
			"MATCH,Proxy",
		},
		TemplateIDs: []string{
			"builtin-clash-general",
			"builtin-singbox-mobile",
			"private-base-template",
		},
		EditorFeatures: []string{
			"inline-validation",
			"drag-sort",
			"yaml-preview",
			"template-injection",
			"conflict-hints",
		},
	}
}

func (s Service) List() ([]RuleSet, error) {
	if s.repo == nil {
		return nil, errors.New("rules repository is not configured")
	}
	return s.repo.ListRuleSets()
}

func (s Service) CreateRuleSet(input CreateRuleSetInput) (RuleSet, error) {
	if s.repo == nil {
		return RuleSet{}, errors.New("rules repository is not configured")
	}
	if result := Validate(input); !result.Valid {
		return RuleSet{}, errors.New(result.Issues[0].Message)
	}
	return s.repo.CreateRuleSet(input)
}

func (s Service) UpdateRuleSet(id string, input CreateRuleSetInput) (RuleSet, error) {
	if s.repo == nil {
		return RuleSet{}, errors.New("rules repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return RuleSet{}, errors.New("rule set id is required")
	}
	if result := Validate(input); !result.Valid {
		return RuleSet{}, errors.New(result.Issues[0].Message)
	}
	return s.repo.UpdateRuleSet(id, input)
}

func (s Service) DeleteRuleSet(id string) error {
	if s.repo == nil {
		return errors.New("rules repository is not configured")
	}
	if strings.TrimSpace(id) == "" {
		return errors.New("rule set id is required")
	}
	return s.repo.DeleteRuleSet(id)
}

func (s Service) Validate(input CreateRuleSetInput) ValidationResult {
	return Validate(input)
}

func Validate(input CreateRuleSetInput) ValidationResult {
	var issues []ValidationIssue
	if strings.TrimSpace(input.Name) == "" {
		issues = append(issues, ValidationIssue{Path: "name", Message: "rule set name is required"})
	}
	seenSortOrders := map[int]bool{}
	for index, rule := range input.Rules {
		path := fmt.Sprintf("rules[%d]", index)
		ruleType := strings.TrimSpace(rule.RuleType)
		if ruleType == "" {
			issues = append(issues, ValidationIssue{Path: path + ".ruleType", Message: "rule type is required"})
			continue
		}
		if !supportedRuleType(ruleType) {
			issues = append(issues, ValidationIssue{Path: path + ".ruleType", Message: "unsupported rule type: " + ruleType})
		}
		if strings.TrimSpace(rule.Policy) == "" {
			issues = append(issues, ValidationIssue{Path: path + ".policy", Message: "rule policy is required"})
		}
		if ruleType != "MATCH" && strings.TrimSpace(rule.Pattern) == "" {
			issues = append(issues, ValidationIssue{Path: path + ".pattern", Message: "rule pattern is required"})
		}
		if ruleType == "MATCH" && strings.TrimSpace(rule.Pattern) != "" {
			issues = append(issues, ValidationIssue{Path: path + ".pattern", Message: "MATCH rules must not include a pattern"})
		}
		if (ruleType == "IP-CIDR" || ruleType == "SRC-IP-CIDR") && strings.TrimSpace(rule.Pattern) != "" {
			if _, _, err := net.ParseCIDR(rule.Pattern); err != nil {
				issues = append(issues, ValidationIssue{Path: path + ".pattern", Message: "CIDR pattern is invalid"})
			}
		}
		if rule.SortOrder < 0 {
			issues = append(issues, ValidationIssue{Path: path + ".sortOrder", Message: "sort order cannot be negative"})
		}
		if rule.SortOrder > 0 {
			if seenSortOrders[rule.SortOrder] {
				issues = append(issues, ValidationIssue{Path: path + ".sortOrder", Message: "sort order is duplicated"})
			}
			seenSortOrders[rule.SortOrder] = true
		}
	}
	return ValidationResult{Valid: len(issues) == 0, Issues: issues}
}

func supportedRuleType(ruleType string) bool {
	switch ruleType {
	case "DOMAIN", "DOMAIN-SUFFIX", "DOMAIN-KEYWORD", "IP-CIDR", "SRC-IP-CIDR", "GEOIP", "MATCH":
		return true
	default:
		return false
	}
}
