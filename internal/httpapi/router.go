package httpapi

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"harborx/internal/features/auth"
	"harborx/internal/features/backups"
	"harborx/internal/features/catalog"
	"harborx/internal/features/certificates"
	"harborx/internal/features/dashboard"
	"harborx/internal/features/dns"
	"harborx/internal/features/nodes"
	"harborx/internal/features/notifications"
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			if err := deps.Users.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			if err := deps.Nodes.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			items, err := deps.Subscriptions.List()
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
				writeJSON(w, http.StatusOK, item)
			case http.MethodDelete:
				if !requireAuth(w, r, deps) {
					return
				}
				if err := deps.Subscriptions.Delete(parts[0]); err != nil {
					writeError(w, http.StatusBadRequest, err)
					return
				}
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
		if r.Method != http.MethodGet {
			writeMethodNotAllowed(w, http.MethodGet)
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.Packages.DeletePackage(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.Packages.DeleteEntitlement(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.Rules.DeleteRuleSet(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.Templates.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.ProxyGroups.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
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
			item, err := deps.Xray.SaveSnapshot(input.TargetKind, input.TargetID)
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
		}
	})

	mux.HandleFunc("/api/v1/xray/snapshots/", func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/api/v1/xray/snapshots/")
		parts := strings.Split(strings.Trim(path, "/"), "/")
		if len(parts) == 2 && parts[1] == "restore" {
			if r.Method != http.MethodPost {
				writeMethodNotAllowed(w, http.MethodPost)
				return
			}
			if !requireAuth(w, r, deps) {
				return
			}
			item, err := deps.Xray.RestoreSnapshot(parts[0])
			if err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
			writeJSON(w, http.StatusOK, item)
			return
		}
		writeError(w, http.StatusNotFound, errors.New("xray snapshot action not found"))
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
				writeJSON(w, http.StatusOK, items)
			case http.MethodPost:
				if !requireAuth(w, r, deps) {
					return
				}
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
			writeJSON(w, http.StatusOK, items)
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
			writeJSON(w, http.StatusOK, items)
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

	mux.HandleFunc("/api/v1/traffic/samples", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			items, err := deps.Traffic.ListSamples(r.URL.Query().Get("scope"), r.URL.Query().Get("scopeId"))
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusCreated, item)
		default:
			writeMethodNotAllowed(w, http.MethodGet, http.MethodPost)
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
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
			if !requireAuth(w, r, deps) {
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
			writeJSON(w, http.StatusOK, item)
		case http.MethodDelete:
			if !requireAuth(w, r, deps) {
				return
			}
			if err := deps.Certificates.Delete(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			writeJSON(w, http.StatusOK, items)
		case http.MethodPost:
			if !requireAuth(w, r, deps) {
				return
			}
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
			if err := deps.Notifications.DeleteChannel(id); err != nil {
				writeError(w, http.StatusBadRequest, err)
				return
			}
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
			writeJSON(w, http.StatusOK, items)
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
		items, err := deps.System.ListSettings()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusOK, items)
	})

	mux.HandleFunc("/api/v1/system/settings/", func(w http.ResponseWriter, r *http.Request) {
		key := strings.TrimPrefix(r.URL.Path, "/api/v1/system/settings/")
		if key == "" {
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

	return withCORS(mux)
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
				"Self-hosted scaffold inspired by miaomiaowuX",
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
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, payload any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{
		"error": err.Error(),
	})
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	return decoder.Decode(target)
}

func writeMethodNotAllowed(w http.ResponseWriter, methods ...string) {
	w.Header().Set("Allow", strings.Join(methods, ", "))
	writeError(w, http.StatusMethodNotAllowed, errors.New("method not allowed"))
}

func requireAuth(w http.ResponseWriter, r *http.Request, deps Dependencies) bool {
	if _, err := deps.Auth.AuthenticateBearer(r.Header.Get("Authorization")); err != nil {
		writeError(w, http.StatusUnauthorized, err)
		return false
	}
	return true
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
