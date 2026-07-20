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

	mux.HandleFunc("/api/v1/users/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Users.Summary())
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

	mux.HandleFunc("/api/v1/nodes/", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete {
			writeMethodNotAllowed(w, http.MethodDelete)
			return
		}
		id := strings.TrimPrefix(r.URL.Path, "/api/v1/nodes/")
		if id == "" {
			writeError(w, http.StatusBadRequest, errors.New("node id is required"))
			return
		}
		if err := deps.Nodes.Delete(id); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		w.WriteHeader(http.StatusNoContent)
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
		if len(parts) != 2 || parts[0] == "" {
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

	mux.HandleFunc("/api/v1/proxy-groups/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.ProxyGroups.Summary())
	})

	mux.HandleFunc("/api/v1/xray/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Xray.Summary())
	})

	mux.HandleFunc("/api/v1/remote/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Remote.Summary())
	})

	mux.HandleFunc("/api/v1/traffic/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Traffic.Summary())
	})

	mux.HandleFunc("/api/v1/certificates/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Certificates.Summary())
	})

	mux.HandleFunc("/api/v1/dns/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.DNS.Summary())
	})

	mux.HandleFunc("/api/v1/notifications/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Notifications.Summary())
	})

	mux.HandleFunc("/api/v1/backups/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.Backups.Summary())
	})

	mux.HandleFunc("/api/v1/system/summary", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, deps.System.Summary())
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
