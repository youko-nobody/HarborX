package app

import (
	"net/http"

	"harborx/internal/config"
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
	"harborx/internal/httpapi"
	"harborx/internal/storage"
)

type App struct {
	Config config.Config
	Router http.Handler
}

func New() (App, error) {
	cfg := config.Load()
	store, err := storage.OpenSQLite(cfg)
	if err != nil {
		return App{}, err
	}

	catalogService := catalog.NewService()
	dashboardService := dashboard.NewService(catalogService)
	authService := auth.NewService()
	usersService := users.NewService()
	nodesService := nodes.NewService(store)
	subscriptionsService := subscriptions.NewService(store)
	rulesService := rules.NewService(store)
	templatesService := templates.NewService(store)
	proxyGroupsService := proxygroups.NewService()
	xrayService := xray.NewService()
	remoteService := remote.NewService()
	trafficService := traffic.NewService()
	certificatesService := certificates.NewService()
	dnsService := dns.NewService()
	notificationsService := notifications.NewService()
	backupsService := backups.NewService()
	systemService := system.NewService()

	router := httpapi.NewRouter(httpapi.Dependencies{
		Catalog:       catalogService,
		Dashboard:     dashboardService,
		Auth:          authService,
		Users:         usersService,
		Nodes:         nodesService,
		Subscriptions: subscriptionsService,
		Rules:         rulesService,
		Templates:     templatesService,
		ProxyGroups:   proxyGroupsService,
		Xray:          xrayService,
		Remote:        remoteService,
		Traffic:       trafficService,
		Certificates:  certificatesService,
		DNS:           dnsService,
		Notifications: notificationsService,
		Backups:       backupsService,
		System:        systemService,
		WebDistDir:    cfg.WebDistDir,
	})

	return App{
		Config: cfg,
		Router: router,
	}, nil
}
