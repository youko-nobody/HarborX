INSERT OR IGNORE INTO users (
    id, username, password_hash, role, status, display_name, email, totp_secret, created_at, updated_at
) VALUES (
    'local-admin',
    'admin',
    '',
    'admin',
    'active',
    'Local Admin',
    '',
    '',
    '2026-07-20T00:00:00Z',
    '2026-07-20T00:00:00Z'
);

INSERT OR IGNORE INTO templates (
    id, name, kind, description, engine, variables_json, content, locked, created_at, updated_at
) VALUES
(
    'builtin-clash-general',
    'Builtin Clash General',
    'builtin',
    'Starter Clash template with proxy groups and rules placeholders.',
    'text-template',
    '["subscription_name","proxy_groups","rules"]',
    'mixed-port: 7890
allow-lan: true
mode: rule
log-level: info
proxies:
{{ .Proxies }}
proxy-groups:
{{ .ProxyGroups }}
rules:
{{ .Rules }}
',
    1,
    '2026-07-20T00:00:00Z',
    '2026-07-20T00:00:00Z'
),
(
    'builtin-singbox-mobile',
    'Builtin Sing-box Mobile',
    'builtin',
    'Starter Sing-box template for mobile clients.',
    'text-template',
    '["subscription_name","outbounds","route_rules"]',
    '{
  "log": { "level": "info" },
  "outbounds": {{ .Outbounds }},
  "route": { "rules": {{ .RouteRules }} }
}
',
    1,
    '2026-07-20T00:00:00Z',
    '2026-07-20T00:00:00Z'
),
(
    'private-base-template',
    'Private Base Template',
    'private',
    'Placeholder custom template to be replaced by your own content later.',
    'text-template',
    '["subscription_name","rules","dns","proxy_groups"]',
    '# Replace this content with your private template
subscription-name: {{ .SubscriptionName }}
dns:
{{ .DNS }}
proxy-groups:
{{ .ProxyGroups }}
rules:
{{ .Rules }}
',
    0,
    '2026-07-20T00:00:00Z',
    '2026-07-20T00:00:00Z'
);
