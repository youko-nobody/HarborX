package rules

import "errors"

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
	return s.repo.CreateRuleSet(input)
}
