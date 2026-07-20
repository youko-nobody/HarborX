package catalog

type Module struct {
	Key         string   `json:"key"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Status      string   `json:"status"`
	Capabilities []string `json:"capabilities"`
}

type Service struct {
	modules []Module
}

func NewService() Service {
	return Service{
		modules: []Module{
			{
				Key:         "auth",
				Name:        "Auth",
				Description: "Login, sessions, API tokens, and two-factor authentication.",
				Status:      "planned",
				Capabilities: []string{"password-login", "session-management", "2fa", "api-tokens"},
			},
			{
				Key:         "users",
				Name:        "Users",
				Description: "Admin and member lifecycle, profile defaults, and permissions.",
				Status:      "planned",
				Capabilities: []string{"user-crud", "roles", "preferences", "limits"},
			},
			{
				Key:         "nodes",
				Name:        "Nodes",
				Description: "Node inventory, inbound and outbound metadata, and tagging.",
				Status:      "planned",
				Capabilities: []string{"node-import", "node-edit", "tags", "health"},
			},
			{
				Key:         "subscriptions",
				Name:        "Subscriptions",
				Description: "Subscription generation, output formats, and user delivery.",
				Status:      "planned",
				Capabilities: []string{"user-subscribe", "format-export", "source-aggregation"},
			},
			{
				Key:         "rules",
				Name:        "Rules Studio",
				Description: "Visual Clash rule editing, validation, and ordering.",
				Status:      "in_progress",
				Capabilities: []string{"visual-editor", "yaml-preview", "validation", "drag-sort"},
			},
			{
				Key:         "templates",
				Name:        "Templates",
				Description: "Built-in private templates and variable-driven rendering.",
				Status:      "in_progress",
				Capabilities: []string{"system-templates", "private-templates", "render-preview"},
			},
			{
				Key:         "proxygroups",
				Name:        "Proxy Groups",
				Description: "Policy group definitions and rule targets.",
				Status:      "planned",
				Capabilities: []string{"group-crud", "fallback", "selector", "url-test"},
			},
			{
				Key:         "xray",
				Name:        "Xray",
				Description: "Config generation, snapshots, and apply planning.",
				Status:      "planned",
				Capabilities: []string{"config-render", "snapshots", "diff", "apply"},
			},
			{
				Key:         "remote",
				Name:        "Remote Servers",
				Description: "Master and agent server orchestration, commands, and status.",
				Status:      "planned",
				Capabilities: []string{"server-registry", "agent-heartbeat", "task-streaming"},
			},
			{
				Key:         "traffic",
				Name:        "Traffic",
				Description: "Usage aggregation, dashboards, and snapshots.",
				Status:      "planned",
				Capabilities: []string{"usage-summary", "history", "server-aggregation"},
			},
			{
				Key:         "certificates",
				Name:        "Certificates",
				Description: "ACME, DNS provider integration, and deploy flows.",
				Status:      "planned",
				Capabilities: []string{"acme", "dns-providers", "auto-renew", "deploy"},
			},
			{
				Key:         "notifications",
				Name:        "Notifications",
				Description: "Telegram and future channel notifications for events and limits.",
				Status:      "planned",
				Capabilities: []string{"telegram", "daily-summary", "alerts"},
			},
			{
				Key:         "backups",
				Name:        "Backups",
				Description: "Export, restore, rotation, and audit logging.",
				Status:      "planned",
				Capabilities: []string{"backup-export", "restore", "retention"},
			},
			{
				Key:         "system",
				Name:        "System Settings",
				Description: "Global defaults, UI behavior, and runtime settings.",
				Status:      "planned",
				Capabilities: []string{"defaults", "themes", "refresh-intervals", "audit"},
			},
		},
	}
}

func (s Service) Modules() []Module {
	return s.modules
}

