package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"harborx/internal/features/audit"
	"harborx/internal/features/auth"
	"harborx/internal/features/backups"
	"harborx/internal/features/catalog"
	"harborx/internal/features/certificates"
	"harborx/internal/features/dashboard"
	"harborx/internal/features/dns"
	"harborx/internal/features/nodes"
	"harborx/internal/features/notifications"
	"harborx/internal/features/ops"
	"harborx/internal/features/packages"
	"harborx/internal/features/proxygroups"
	"harborx/internal/features/remote"
	"harborx/internal/features/rules"
	"harborx/internal/features/subscriptions"
	"harborx/internal/features/system"
	"harborx/internal/features/templates"
	"harborx/internal/features/traffic"
	"harborx/internal/features/users"
	"harborx/internal/features/xray"
)

type Dependencies struct {
	Catalog       catalog.Service
	Dashboard     dashboard.Service
	Auth          auth.Service
	Audit         audit.Service
	Users         users.Service
	Nodes         nodes.Service
	Subscriptions subscriptions.Service
	Rules         rules.Service
	Templates     templates.Service
	ProxyGroups   proxygroups.Service
	Xray          xray.Service
	Remote        remote.Service
	Traffic       traffic.Service
	Certificates  certificates.Service
	DNS           dns.Service
	Notifications notifications.Service
	Ops           ops.Service
	Packages      packages.Service
	Backups       backups.Service
	System        system.Service
	WebDistDir    string
}

func NewRouter(deps Dependencies) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/api/v1/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"status":  "ok",
			"service": "harborx",
		})
	})

	mux.HandleFunc("/api/v1/catalog/modules", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Catalog.Modules())
	})

	mux.HandleFunc("/api/v1/dashboard/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Dashboard.Summary())
	})

	mux.HandleFunc("/api/v1/audit/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Audit.Summary())
	})

	mux.HandleFunc("/api/v1/audit/events", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		limit := 100
		items, err := deps.Audit.List(limit)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	mux.HandleFunc("/api/v1/auth/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Auth.Summary())
	})

	mux.HandleFunc("/api/v1/auth/login", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		var input auth.LoginInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result, err := deps.Auth.Login(input)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		_ = deps.Audit.Record(audit.CreateEntryInput{
			ActorID:       result.User.ID,
			ActorUsername: result.User.Username,
			Action:        "auth.login",
			ResourceType:  "session",
			IP:            clientIP(r),
		})
		writeJSON(w, http.StatusOK, result)
	})

	mux.HandleFunc("/api/v1/users/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Users.Summary())
	})

	mux.HandleFunc("/api/v1/users", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if !requireAuth(w, r, deps) {
				return
			}
			items, err := deps.Users.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input users.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Users.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{
				ActorID:       op.id,
				ActorUsername: op.username,
				Action:        "user.create",
				ResourceType:  "user",
				ResourceID:    item.ID,
				IP:            clientIP(r),
			})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/users/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/users/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("user id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			var input users.UpdateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Users.Update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Users.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "user.delete", ResourceType: "user", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/nodes/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Nodes.Summary())
	})

	mux.HandleFunc("/api/v1/nodes", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Nodes.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input nodes.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Nodes.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "node.create", ResourceType: "node", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/nodes/import", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		var input nodes.ImportInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		result, err := deps.Nodes.Import(input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, result)
	})

	mux.HandleFunc("/api/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("node id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			var input nodes.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Nodes.Update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Nodes.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "node.delete", ResourceType: "node", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/subscriptions/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Subscriptions.Summary())
	})

	mux.HandleFunc("/api/v1/subscriptions", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			if !requireAuth(w, r, deps) {
				return
			}
			items, err := deps.Subscriptions.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input subscriptions.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Subscriptions.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{
				ActorID:       op.id,
				ActorUsername: op.username,
				Action:        "subscription.create",
				ResourceType:  "subscription",
				ResourceID:    item.ID,
				IP:            clientIP(r),
			})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/subscriptions/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/subscriptions/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, errors.New("subscription id is required"))
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodPut:
				if !requireAuth(w, r, deps) {
					return
				}
				opSub := authenticatedOperator(deps, r.Header.Get("Authorization"))
				var input subscriptions.CreateInput
				if err := decodeJSON(r, &input); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				item, err := deps.Subscriptions.Update(parts[0], input)
				if err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opSub.id, ActorUsername: opSub.username, Action: "subscription.update", ResourceType: "subscription", ResourceID: parts[0], IP: clientIP(r)})
				writeJSON(w, http.StatusOK, item)
			case http.MethodDelete:
				if !requireAuth(w, r, deps) {
					return
				}
				op := authenticatedOperator(deps, r.Header.Get("Authorization"))
				if err := deps.Subscriptions.Delete(parts[0]); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "subscription.delete", ResourceType: "subscription", ResourceID: parts[0], IP: clientIP(r)})
				w.WriteHeader(http.StatusNoContent)
			default:
				writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
			}
			return
		}

		if len(parts) != 2 {
			writeError(w, http.StatusBadRequest, errors.New("subscription action path must be /api/v1/subscriptions/{id}/preview or /download"))
			return
		}

		if parts[1] == "token-rotate" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			rotated, err := deps.Subscriptions.RotateAccessToken(parts[0])
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{
				ActorID:       op.id,
				ActorUsername: op.username,
				Action:        "subscription.rotate-token",
				ResourceType:  "subscription",
				ResourceID:    parts[0],
				IP:            clientIP(r),
			})
			writeJSON(w, http.StatusOK, rotated)
			return
		}
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}

		bearerToken := r.Header.Get("Authorization")
		accessToken := r.URL.Query().Get("token")
		if !requireAuth(w, r, deps) && !deps.Subscriptions.CanAuthenticate(parts[0], bearerToken, accessToken) {
			writeError(w, http.StatusUnauthorized, errors.New("invalid token"))
			return
		}

		rendered, err := deps.Subscriptions.Render(parts[0])
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}

		switch parts[1] {
		case "preview":
			writeJSON(w, http.StatusOK, rendered)
		case "download":
			w.Header().Set("Content-Type", rendered.ContentType)
			w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", rendered.FileName))
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(rendered.Content))
		default:
			writeError(w, http.StatusNotFound, errors.New("subscription action not found"))
		}
	})

	mux.HandleFunc("/api/v1/packages/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Packages.Summary())
	})

	mux.HandleFunc("/api/v1/packages", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Packages.ListPackages()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opPkg := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input packages.CreatePackageInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Packages.CreatePackage(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPkg.id, ActorUsername: opPkg.username, Action: "package.create", ResourceType: "package", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/packages/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/packages/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("package id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opPkg := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input packages.CreatePackageInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Packages.UpdatePackage(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPkg.id, ActorUsername: opPkg.username, Action: "package.update", ResourceType: "package", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opPkg := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Packages.DeletePackage(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPkg.id, ActorUsername: opPkg.username, Action: "package.delete", ResourceType: "package", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/entitlements", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Packages.ListEntitlements(r.URL.Query().Get("userId"))
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opEnt := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input packages.CreateEntitlementInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Packages.CreateEntitlement(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opEnt.id, ActorUsername: opEnt.username, Action: "entitlement.create", ResourceType: "entitlement", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/entitlements/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/entitlements/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("entitlement id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opEnt := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input packages.CreateEntitlementInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Packages.UpdateEntitlement(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opEnt.id, ActorUsername: opEnt.username, Action: "entitlement.update", ResourceType: "entitlement", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opEnt := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Packages.DeleteEntitlement(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opEnt.id, ActorUsername: opEnt.username, Action: "entitlement.delete", ResourceType: "entitlement", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/rules/bootstrap", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Rules.Bootstrap())
	})

	mux.HandleFunc("/api/v1/rulesets", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Rules.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opRule := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input rules.CreateRuleSetInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Rules.CreateRuleSet(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opRule.id, ActorUsername: opRule.username, Action: "ruleset.create", ResourceType: "ruleset", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/rulesets/validate", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		var input rules.CreateRuleSetInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, deps.Rules.Validate(input))
	})

	mux.HandleFunc("/api/v1/rulesets/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/rulesets/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("rule set id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opRule := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input rules.CreateRuleSetInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Rules.UpdateRuleSet(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opRule.id, ActorUsername: opRule.username, Action: "ruleset.update", ResourceType: "ruleset", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opRule := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Rules.DeleteRuleSet(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opRule.id, ActorUsername: opRule.username, Action: "ruleset.delete", ResourceType: "ruleset", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/templates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Templates.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opTpl := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input templates.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Templates.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opTpl.id, ActorUsername: opTpl.username, Action: "template.create", ResourceType: "template", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/templates/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/templates/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("template id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opTpl := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input templates.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Templates.Update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opTpl.id, ActorUsername: opTpl.username, Action: "template.update", ResourceType: "template", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opTpl := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Templates.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opTpl.id, ActorUsername: opTpl.username, Action: "template.delete", ResourceType: "template", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/proxy-groups/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.ProxyGroups.Summary())
	})

	mux.HandleFunc("/api/v1/proxy-groups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.ProxyGroups.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opPG := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input proxygroups.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.ProxyGroups.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPG.id, ActorUsername: opPG.username, Action: "proxygroup.create", ResourceType: "proxy-group", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/proxy-groups/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/proxy-groups/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("proxy group id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opPG := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input proxygroups.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.ProxyGroups.Update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPG.id, ActorUsername: opPG.username, Action: "proxygroup.update", ResourceType: "proxy-group", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opPG := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.ProxyGroups.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opPG.id, ActorUsername: opPG.username, Action: "proxygroup.delete", ResourceType: "proxy-group", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/xray/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Xray.Summary())
	})

	mux.HandleFunc("/api/v1/xray/preview", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		preview, err := deps.Xray.Preview()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, preview)
	})

	mux.HandleFunc("/api/v1/xray/snapshots", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Xray.ListSnapshots(r.URL.Query().Get("targetKind"), r.URL.Query().Get("targetId"))
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			var input struct {
				TargetKind string `json:"targetKind"`
				TargetID   string `json:"targetId"`
			}
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			opXray := authenticatedOperator(deps, r.Header.Get("Authorization"))
			item, err := deps.Xray.SaveSnapshot(input.TargetKind, input.TargetID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opXray.id, ActorUsername: opXray.username, Action: "xray.save-snapshot", ResourceType: "xray-snapshot", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/xray/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/xray/snapshots/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, errors.New("xray snapshot id is required"))
			return
		}
		// Restrict snapshot ids to the characters used by our UUIDs. Reject any
		// other characters (notably "../") so a crafted id cannot be interpreted
		// as a relative path by the storage layer.
		for _, c := range parts[0] {
			if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'z') || c == '-') {
				writeError(w, http.StatusBadRequest, errors.New("xray snapshot id is required"))
				return
			}
		}
		if len(parts) == 2 && parts[1] == "restore" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			opXray := authenticatedOperator(deps, r.Header.Get("Authorization"))
			item, err := deps.Xray.RestoreSnapshot(parts[0])
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opXray.id, ActorUsername: opXray.username, Action: "xray.restore-snapshot", ResourceType: "xray-snapshot", ResourceID: parts[0], IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
			return
		}
		writeError(w, http.StatusNotFound, errors.New("xray snapshot action not found"))
	})

	mux.HandleFunc("/api/v1/xray/profiles", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Xray.ListProfiles()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opXray := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input xray.CreateProfileInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Xray.CreateProfile(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opXray.id, ActorUsername: opXray.username, Action: "xray.create-profile", ResourceType: "xray-profile", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/xray/profiles/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/xray/profiles/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, errors.New("xray profile id is required"))
			return
		}
		if len(parts) == 2 && parts[1] == "apply" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			var input xray.ApplyInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			input.ProfileID = parts[0]
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			item, err := deps.Xray.Apply(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "xray.apply-profile", ResourceType: "xray-profile", ResourceID: parts[0], IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
			return
		}

		if len(parts) != 1 {
			writeError(w, http.StatusNotFound, errors.New("xray profile action not found"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			opXray := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input xray.CreateProfileInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Xray.UpdateProfile(parts[0], input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opXray.id, ActorUsername: opXray.username, Action: "xray.update-profile", ResourceType: "xray-profile", ResourceID: parts[0], IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opXray := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Xray.DeleteProfile(parts[0]); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opXray.id, ActorUsername: opXray.username, Action: "xray.delete-profile", ResourceType: "xray-profile", ResourceID: parts[0], IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/remote/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Remote.Summary())
	})

	mux.HandleFunc("/api/v1/remote/servers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Remote.ListServers()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input remote.CreateServerInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			enrollment, err := deps.Remote.CreateServer(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "remote.create-server", ResourceType: "remote-server", ResourceID: enrollment.Server.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, enrollment)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/remote/servers/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/remote/servers/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, errors.New("remote server id is required"))
			return
		}

		if len(parts) == 1 {
			switch r.Method {
			case http.MethodPut:
				if !requireAuth(w, r, deps) {
					return
				}
				var input remote.UpdateServerInput
				if err := decodeJSON(r, &input); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				item, err := deps.Remote.UpdateServer(parts[0], input)
				if err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, item)
			case http.MethodDelete:
				if !requireAuth(w, r, deps) {
					return
				}
				if err := deps.Remote.DeleteServer(parts[0]); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				w.WriteHeader(http.StatusNoContent)
			default:
				writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
			}
			return
		}

		if len(parts) == 2 && parts[1] == "tasks" {
			switch r.Method {
			case http.MethodGet:
				items, err := deps.Remote.ListTasks(parts[0])
				if err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
			case http.MethodPost:
				if !requireAuth(w, r, deps) {
					return
				}
				opRemote := authenticatedOperator(deps, r.Header.Get("Authorization"))
				var input remote.CreateTaskInput
				if err := decodeJSON(r, &input); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				item, err := deps.Remote.CreateTask(parts[0], input)
				if err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
				_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opRemote.id, ActorUsername: opRemote.username, Action: "remote.create-task", ResourceType: "remote-task", ResourceID: item.ID, Detail: map[string]any{"serverId": parts[0], "taskKind": item.TaskKind}, IP: clientIP(r)})
				writeJSON(w, http.StatusCreated, item)
			default:
				writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
			}
			return
		}

		if len(parts) == 2 && parts[1] == "logs" {
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w, http.MethodGet)
				return
			}
			items, err := deps.Remote.ListAgentLogs(parts[0])
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
			return
		}

		if len(parts) == 4 && parts[1] == "tasks" && parts[3] == "logs" {
			if r.Method != http.MethodGet {
				writeMethodNotAllowed(w, http.MethodGet)
				return
			}
			items, err := deps.Remote.ListTaskLogs(parts[0], parts[2])
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
			return
		}

		writeError(w, http.StatusNotFound, errors.New("remote server action not found"))
	})

	mux.HandleFunc("/api/v1/agent/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		var input remote.AgentHeartbeatInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		server, err := deps.Remote.AgentHeartbeat(agentTokenFromRequest(r), input)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusOK, server)
	})

	mux.HandleFunc("/api/v1/agent/tasks/claim", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		claim, err := deps.Remote.AgentClaimTask(agentTokenFromRequest(r))
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusOK, claim)
	})

	mux.HandleFunc("/api/v1/agent/logs", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		var input remote.CreateAgentLogInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := deps.Remote.AgentLog(agentTokenFromRequest(r), input)
		if err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("/api/v1/agent/traffic", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		var input remote.AgentTrafficSampleInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := deps.Remote.AgentTrafficSample(agentTokenFromRequest(r), input); err != nil {
			writeError(w, http.StatusUnauthorized, err)
			return
		}
		writeJSON(w, http.StatusCreated, map[string]any{"ok": true})
	})

	mux.HandleFunc("/api/v1/agent/tasks/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		taskID := strings.TrimPrefix(r.URL.Path, "/api/v1/agent/tasks/")
		if taskID == "" {
			writeError(w, http.StatusBadRequest, errors.New("remote task id is required"))
			return
		}
		var input remote.AgentTaskUpdateInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := deps.Remote.AgentUpdateTask(agentTokenFromRequest(r), taskID, input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusOK, item)
	})

	mux.HandleFunc("/api/v1/traffic/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Traffic.Summary())
	})

	mux.HandleFunc("/api/v1/traffic/rollups", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		items, err := deps.Traffic.Rollups(r.URL.Query().Get("scope"), r.URL.Query().Get("scopeId"))
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
	})

	mux.HandleFunc("/api/v1/traffic/samples", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Traffic.ListSamples(r.URL.Query().Get("scope"), r.URL.Query().Get("scopeId"))
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opTraffic := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input traffic.CreateSampleInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Traffic.CreateSample(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opTraffic.id, ActorUsername: opTraffic.username, Action: "traffic.create-sample", ResourceType: "traffic-sample", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/ops/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Ops.Summary())
	})

	mux.HandleFunc("/api/v1/ops/resources", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Ops.List(r.URL.Query().Get("kind"))
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input ops.CreateResourceInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Ops.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "ops.create-resource", ResourceType: "ops-resource", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/ops/resources/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/ops/resources/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 0 || parts[0] == "" {
			writeError(w, http.StatusBadRequest, errors.New("ops resource id is required"))
			return
		}

		if len(parts) == 2 && parts[1] == "execute" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input ops.ExecuteInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Ops.Execute(parts[0], input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "ops.execute", ResourceType: "ops-resource", ResourceID: parts[0], Detail: map[string]any{"taskKind": item.TaskKind, "dryRun": item.DryRun}, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
			return
		}

		if len(parts) != 1 {
			writeError(w, http.StatusNotFound, errors.New("ops resource action not found"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input ops.CreateResourceInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Ops.Update(parts[0], input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "ops.update-resource", ResourceType: "ops-resource", ResourceID: parts[0], IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			op := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Ops.Delete(parts[0]); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: op.id, ActorUsername: op.username, Action: "ops.delete-resource", ResourceType: "ops-resource", ResourceID: parts[0], IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/certificates/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Certificates.Summary())
	})

	mux.HandleFunc("/api/v1/certificates", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Certificates.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			opCert := requireOperator(w, r, deps)
			if opCert.id == "" {
				return
			}
			var input certificates.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Certificates.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opCert.id, ActorUsername: opCert.username, Action: "certificate.create", ResourceType: "certificate", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/certificates/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/certificates/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("certificate id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			opCert := requireOperator(w, r, deps)
			if opCert.id == "" {
				return
			}
			var input certificates.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Certificates.Update(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opCert.id, ActorUsername: opCert.username, Action: "certificate.update", ResourceType: "certificate", ResourceID: id, IP: clientIP(r)})
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			opCert := requireOperator(w, r, deps)
			if opCert.id == "" {
				return
			}
			if err := deps.Certificates.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opCert.id, ActorUsername: opCert.username, Action: "certificate.delete", ResourceType: "certificate", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/dns/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.DNS.Summary())
	})

	mux.HandleFunc("/api/v1/dns/providers", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.DNS.ListProviders()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opDNS := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input dns.CreateProviderInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.DNS.CreateProvider(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opDNS.id, ActorUsername: opDNS.username, Action: "dns.create-provider", ResourceType: "dns-provider", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/dns/providers/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/dns/providers/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("dns provider id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			var input dns.CreateProviderInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.DNS.UpdateProvider(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.DNS.DeleteProvider(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/notifications/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Notifications.Summary())
	})

	mux.HandleFunc("/api/v1/notifications/channels", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Notifications.ListChannels()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			opNotify := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input notifications.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Notifications.CreateChannel(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opNotify.id, ActorUsername: opNotify.username, Action: "notification.create-channel", ResourceType: "notification-channel", ResourceID: item.ID, IP: clientIP(r)})
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/notifications/channels/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/notifications/channels/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 2 && parts[1] == "test" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			opNotify := authenticatedOperator(deps, r.Header.Get("Authorization"))
			var input struct {
				Message string `json:"message"`
			}
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			if err := deps.Notifications.TestChannel(parts[0], input.Message); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opNotify.id, ActorUsername: opNotify.username, Action: "notification.test-channel", ResourceType: "notification-channel", ResourceID: parts[0], IP: clientIP(r)})
			writeJSON(w, http.StatusOK, map[string]any{"ok": true})
			return
		}

		id := strings.TrimSpace(path)
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("notification channel id is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			var input notifications.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Notifications.UpdateChannel(id, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			opNotify := authenticatedOperator(deps, r.Header.Get("Authorization"))
			if err := deps.Notifications.DeleteChannel(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			_ = deps.Audit.Record(audit.CreateEntryInput{ActorID: opNotify.id, ActorUsername: opNotify.username, Action: "notification.delete-channel", ResourceType: "notification-channel", ResourceID: id, IP: clientIP(r)})
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/api/v1/backups/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Backups.Summary())
	})

	mux.HandleFunc("/api/v1/backups", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Backups.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
			var input backups.CreateInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.Backups.Create(input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/backups/export", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			writeMethodNotAllowed(w, http.MethodPost)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		var input backups.ExportInput
		if err := decodeJSON(r, &input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		item, err := deps.Backups.ExportDatabase(input)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusCreated, item)
	})

	mux.HandleFunc("/api/v1/backups/", func(w http.ResponseWriter, r *http.Request) {
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/backups/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("backup id is required"))
			return
		}
		if r.Method != http.MethodDelete {
			writeMethodNotAllowed(w, http.MethodDelete)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		if err := deps.Backups.Delete(id); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("/api/v1/system/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.System.Summary())
	})

	mux.HandleFunc("/api/v1/system/settings", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
			return
		}
		if !requireAuth(w, r, deps) {
			return
		}
		items, err := deps.System.ListSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, limitSlice(items, r.URL.Query().Get("limit")))
	})

	mux.HandleFunc("/api/v1/system/settings/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/system/settings/")
		if key == "" || strings.ContainsRune(key, '/') {
			writeError(w, http.StatusBadRequest, errors.New("system setting key is required"))
			return
		}
		switch r.Method {
		case http.MethodPut:
			if !requireAuth(w, r, deps) {
				return
			}
			var input system.UpsertSettingInput
			if err := decodeJSON(r, &input); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			item, err := deps.System.UpsertSetting(key, input)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.System.DeleteSetting(key); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			w.WriteHeader(http.StatusNoContent)
		default:
			writeMethodNotAllowed(w, http.MethodPut, http.MethodDelete)
		}
	})

	mux.HandleFunc("/", frontendHandler(deps.WebDistDir))

	return withCORS(withSecurityHeaders(mux))
}

func withSecurityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func frontendHandler(distDir string) http.HandlerFunc {
	fileServer := http.FileServer(http.Dir(distDir))
	return func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/" && !directoryExists(distDir) {
			writeJSON(w, http.StatusOK, map[string]any{
				"name": "HarborX API",
				"notes": []string{
					"Frontend build not found. Run npm run build in the web directory.",
					"API endpoints are available under /api/v1.",
				},
			})
			return
		}

		requestPath := strings.TrimPrefix(filepath.Clean(r.URL.Path), string(filepath.Separator))
		fullPath := filepath.Join(distDir, requestPath)
		if info, err := os.Stat(fullPath); err == nil && !info.IsDir() {
			fileServer.ServeHTTP(w, r)
			return
		}

		indexPath := filepath.Join(distDir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			http.ServeFile(w, r, indexPath)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"name": "HarborX API",
			"notes": []string{
				"Self-hosted Xray and subscription control plane",
				"No license or pro gating is included",
				"Frontend build not found. Run npm run build in the web directory.",
			},
		})
	}
}

func directoryExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func withCORS(next http.Handler) http.Handler {
	// Optional origin allow-list via HARBORX_CORS_ORIGINS (comma-separated).
	// When unset, the behaviour is permissive (accept any origin) to preserve
	// compatibility with existing self-hosted deployments. Set this in
	// production to restrict cross-origin access.
	allowedOrigins := []string{}
	if raw := os.Getenv("HARBORX_CORS_ORIGINS"); raw != "" {
		for _, part := range strings.Split(raw, ",") {
			trimmed := strings.TrimSpace(part)
			if trimmed != "" {
				allowedOrigins = append(allowedOrigins, trimmed)
			}
		}
	}
	allowAll := len(allowedOrigins) == 0
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		origin := r.Header.Get("Origin")
		if allowAll {
			w.Header().Set("Access-Control-Allow-Origin", "*")
		} else if origin != "" && contains(allowedOrigins, origin) {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
		}
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-HarborX-Agent-Token")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func contains(items []string, value string) bool {
	for _, item := range items {
		if item == value {
			return true
		}
	}
	return false
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": sanitizeError(err.Error()),
	})
}

// sanitizeError redacts internal details from error messages before they
// reach the HTTP response. Database engines, the filesystem, and the Xray
// core tend to include paths, table names, and SQL in their error strings;
// echoing those back lets an attacker fingerprint the deployment.
func sanitizeError(msg string) string {
	lower := strings.ToLower(msg)
	redactSignals := []string{
		"database", "sqlite", "locked", "primary key", "foreign key",
		"violates", "constraint", "index", "table ", "no such",
		"no such file", "permission denied", "open ", "read ",
	}
	for _, signal := range redactSignals {
		if strings.Contains(lower, signal) {
			switch statusFromMessage(lower) {
			case http.StatusConflict:
				return "resource conflict"
			default:
				return "internal error"
			}
		}
	}
	return msg
}

func statusFromMessage(lower string) int {
	if strings.Contains(lower, "conflict") ||
		strings.Contains(lower, "primary key") ||
		strings.Contains(lower, "unique") ||
		strings.Contains(lower, "violates") ||
		strings.Contains(lower, "duplicate") {
		return http.StatusConflict
	}
	return 0
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	// Cap the request body so a single malformed or oversized payload cannot
	// exhaust process memory. Most real payloads are well under 1 KiB.
	body, err := io.ReadAll(io.LimitReader(r.Body, 4<<20))
	if err != nil {
		return err
	}
	if len(body) > 4<<20 {
		return errors.New("request body is too large")
	}
	decoder := json.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

// limitSlice caps a result list by the optional ?limit query param so a single
// List endpoint cannot be used to pull an unbounded number of rows into memory.
// Default 100, hard cap 500.
func limitSlice[T any](items []T, rawLimit string) []T {
	n := 100
	if rawLimit != "" {
		parsed, err := strconv.Atoi(rawLimit)
		if err == nil && parsed > 0 {
			n = parsed
		}
	}
	if n > 500 {
		n = 500
	}
	if len(items) <= n {
		return items
	}
	return items[:n]
}

func requireAuth(w http.ResponseWriter, r *http.Request, deps Dependencies) bool {
	if _, err := deps.Auth.AuthenticateBearer(r.Header.Get("Authorization")); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return false
	}
	return true
}

type operatorInfo struct {
	id       string
	username string
}

func authenticatedOperator(deps Dependencies, header string) operatorInfo {
	user, err := deps.Auth.AuthenticateBearer(header)
	if err != nil {
		return operatorInfo{}
	}
	return operatorInfo{id: user.ID, username: user.Username}
}

// requireOperator is like requireAuth but additionally rejects any user whose
// role is not "admin". Non-admin (member) users are logged in but are not
// permitted to mutate security-sensitive resources such as certificates,
// DNS providers, system settings, or remote servers.
func requireOperator(w http.ResponseWriter, r *http.Request, deps Dependencies) operatorInfo {
	if _, err := deps.Auth.AuthenticateBearer(r.Header.Get("Authorization")); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return operatorInfo{}
	}
	user, err := deps.Auth.AuthenticateBearer(r.Header.Get("Authorization"))
	if err != nil || user.Role != "admin" {
		writeError(w, http.StatusForbidden, errors.New("admin role required"))
		return operatorInfo{}
	}
	return operatorInfo{id: user.ID, username: user.Username}
}

func agentTokenFromRequest(r *http.Request) string {
	if token := strings.TrimSpace(r.Header.Get("X-HarborX-Agent-Token")); token != "" {
		return token
	}
	const prefix = "Bearer "
	header := r.Header.Get("Authorization")
	if strings.HasPrefix(header, prefix) {
		return strings.TrimSpace(strings.TrimPrefix(header, prefix))
	}
	return ""
}
