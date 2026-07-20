mixed-port: 7890
allow-lan: true
mode: rule
log-level: info
ipv6: true
proxies:
{{ .Proxies }}
proxy-groups:
{{ .ProxyGroups }}
rules:
{{ .Rules }}

