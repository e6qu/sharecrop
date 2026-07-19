package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/e6qu/sharecrop/internal/agent"
	"github.com/e6qu/sharecrop/internal/app"
	"github.com/e6qu/sharecrop/internal/assets"
	"github.com/e6qu/sharecrop/internal/audit"
	"github.com/e6qu/sharecrop/internal/auth"
	"github.com/e6qu/sharecrop/internal/contracts"
	"github.com/e6qu/sharecrop/internal/db"
	httpserver "github.com/e6qu/sharecrop/internal/http"
	"github.com/e6qu/sharecrop/internal/ledger"
	"github.com/e6qu/sharecrop/internal/mcp"
	"github.com/e6qu/sharecrop/internal/notification"
	"github.com/e6qu/sharecrop/internal/openapi"
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/orgcred"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
	"github.com/e6qu/sharecrop/internal/wasibridge/gen"
	"github.com/e6qu/sharecrop/internal/wasibridge/httpbridge"
	"github.com/e6qu/sharecrop/internal/wasibridge/rpc"
	"github.com/e6qu/sharecrop/internal/wasibridge/storehost"
	"github.com/e6qu/sharecrop/internal/wasiguest"
	"github.com/e6qu/sharecrop/web"
)

func main() {
	os.Exit(run(context.Background(), os.Args, os.Stdout, os.Stderr))
}

func run(ctx context.Context, args []string, stdout io.Writer, stderr io.Writer) int {
	logger := slog.New(slog.NewTextHandler(stderr, &slog.HandlerOptions{}))
	if len(args) > 1 && args[1] == "generate" {
		return runGenerate(args[2:], stdout, logger)
	}
	// wasi-precompile is a build-time step that populates the wazero cache; it
	// needs no database or runtime config, so it runs before LoadConfig.
	if len(args) > 1 && args[1] == "wasi-precompile" {
		return runWASIPrecompile(ctx, args[2:], logger)
	}
	if len(args) > 1 && args[1] == "migrate" {
		cfgResult := app.LoadMigrationConfig()
		cfg, loaded := cfgResult.(app.MigrationConfigLoaded)
		if !loaded {
			rejected := cfgResult.(app.MigrationConfigRejected)
			logger.Error("load migration config", "reason", rejected.Reason)
			return 2
		}
		return runMigrate(ctx, args[2:], cfg.Value, stdout, logger)
	}

	cfgResult := app.LoadConfig()
	cfg, loaded := cfgResult.(app.ConfigLoaded)
	if !loaded {
		rejected := cfgResult.(app.ConfigRejected)
		logger.Error("load config", "reason", rejected.Reason)
		return 2
	}

	if len(args) > 1 {
		switch args[1] {
		case "serve":
			return runServe(ctx, cfg.Value, logger)
		case "mcp":
			return runMCPStdio(ctx, cfg.Value, stdout, logger)
		default:
			_, _ = fmt.Fprintf(stderr, "unknown command: %s\n", args[1])
			return 2
		}
	}

	return runServe(ctx, cfg.Value, logger)
}

func runGenerate(args []string, stdout io.Writer, logger *slog.Logger) int {
	if len(args) != 1 {
		_, _ = fmt.Fprintln(stdout, "usage: sharecrop generate elm-contracts|openapi|wasi-bridge")
		return 2
	}
	switch args[0] {
	case "elm-contracts":
		return runGenerateElmContracts(stdout, logger)
	case "openapi":
		return runGenerateOpenAPI(stdout, logger)
	case "wasi-bridge":
		return runGenerateWASIBridge(stdout, logger)
	default:
		_, _ = fmt.Fprintln(stdout, "usage: sharecrop generate elm-contracts|openapi|wasi-bridge")
		return 2
	}
}

// runGenerateWASIBridge regenerates the audit store's WASI bridge
// (internal/wasibridge/auditbridge/bridge_gen.go) from the audit.Store
// interface, so the per-method plumbing tracks the interface. check-wasi-bridge
// runs this and diffs, failing CI if the committed bridge has drifted.
func runGenerateWASIBridge(stdout io.Writer, logger *slog.Logger) int {
	for _, target := range gen.Targets() {
		sources, err := readGoPackageSources(target.SourceDir)
		if err != nil {
			logger.Error("read store package sources", "store", target.Key, "error", err)
			return 1
		}

		source, err := gen.Generate(sources, target.Key)
		if err != nil {
			logger.Error("generate wasi bridge", "store", target.Key, "error", err)
			return 1
		}

		if err := os.WriteFile(target.OutputPath, []byte(source), 0o644); err != nil {
			logger.Error("write wasi bridge", "store", target.Key, "error", err)
			return 1
		}
	}

	_, _ = fmt.Fprintln(stdout, "wasi bridge generated")
	return 0
}

func runGenerateElmContracts(stdout io.Writer, logger *slog.Logger) int {
	filesResult := contracts.GenerateElmFiles(contracts.Modules())
	filesGenerated, filesMatched := filesResult.(contracts.ElmFilesGenerated)
	if !filesMatched {
		rejected := filesResult.(contracts.ElmFilesRejected)
		logger.Error("generate elm contracts", "reason", rejected.Reason)
		return 1
	}

	writeResult := contracts.WriteElmFiles(filesGenerated.Files)
	if _, written := writeResult.(contracts.ElmFilesWritten); !written {
		rejected := writeResult.(contracts.WriteElmFilesRejected)
		logger.Error("write elm contracts", "reason", rejected.Reason)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, "elm contracts generated")
	return 0
}

func runGenerateOpenAPI(stdout io.Writer, logger *slog.Logger) int {
	sources, err := readGoPackageSources("internal/http")
	if err != nil {
		logger.Error("read http package sources", "error", err)
		return 1
	}

	extractResult := openapi.Extract(sources)
	extracted, extractedMatched := extractResult.(openapi.Extracted)
	if !extractedMatched {
		rejected := extractResult.(openapi.ExtractionRejected)
		logger.Error("extract openapi routes", "reason", rejected.Reason)
		return 1
	}

	document := openapi.Generate(extracted.Routes, extracted.Structs)
	writeResult := openapi.Write(document, "docs/openapi.json")
	if _, written := writeResult.(openapi.Written); !written {
		rejected := writeResult.(openapi.WriteRejected)
		logger.Error("write openapi document", "reason", rejected.Reason)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, "openapi document generated")
	return 0
}

// readGoPackageSources reads every non-test Go source file in dir, keyed by
// path, so openapi.Extract can parse the package without depending on the
// build system to enumerate files.
func readGoPackageSources(dir string) (map[string][]byte, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	sources := make(map[string][]byte, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		content, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}
		sources[path] = content
	}
	return sources, nil
}

func runMigrate(ctx context.Context, args []string, cfg app.MigrationConfig, stdout io.Writer, logger *slog.Logger) int {
	if len(args) != 1 || args[0] != "up" {
		_, _ = fmt.Fprintln(stdout, "usage: sharecrop migrate up")
		return 2
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("open database", "error", err)
		return 1
	}
	defer pool.Close()

	if err := db.MigrateUp(ctx, pool, cfg.MigrationsDir()); err != nil {
		logger.Error("run migrations", "error", err)
		return 1
	}

	_, _ = fmt.Fprintln(stdout, "migrations applied")
	return 0
}

func runMCPStdio(ctx context.Context, cfg app.Config, stdout io.Writer, logger *slog.Logger) int {
	rawToken := os.Getenv("SHARECROP_AGENT_TOKEN")

	pool, err := db.Open(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("open database", "error", err)
		return 1
	}
	defer pool.Close()
	if err := db.VerifyMigrationsCurrent(ctx, pool, cfg.MigrationsDir()); err != nil {
		logger.Error("verify database migrations", "error", err)
		return 1
	}

	agentService := agent.NewService(db.NewAgentStore(pool))
	orgCredentialService := orgcred.NewService(db.NewOrgCredentialStore(pool))

	var subject auth.Subject
	var callerCredential mcp.CallerCredential
	if orgcred.HasSecretPrefix(rawToken) {
		secretResult := orgcred.ParseSecretPlain(rawToken)
		secret, secretMatched := secretResult.(orgcred.SecretPlainAccepted)
		if !secretMatched {
			logger.Error("organization credential", "reason", secretResult.(orgcred.SecretPlainRejected).Reason.Description())
			return 2
		}
		verifyResult := orgCredentialService.Verify(ctx, secret.Value)
		verified, verifiedMatched := verifyResult.(orgcred.CredentialVerified)
		if !verifiedMatched {
			logger.Error("verify organization credential", "reason", verifyResult.(orgcred.VerifyRejected).Reason.Description())
			return 1
		}
		subject = verified.Subject
		callerCredential = mcp.CallerCredential{Scopes: verified.Credential.Scopes}
	} else {
		secretResult := agent.ParseSecretPlain(rawToken)
		secret, secretMatched := secretResult.(agent.SecretPlainAccepted)
		if !secretMatched {
			logger.Error("agent credential", "reason", "SHARECROP_AGENT_TOKEN is required and must be a valid agent or organization credential")
			return 2
		}
		verifyResult := agentService.Verify(ctx, secret.Value)
		verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
		if !verifiedMatched {
			logger.Error("verify agent credential", "reason", verifyResult.(agent.VerifyRejected).Reason.Description())
			return 1
		}
		subject = verified.Subject
		callerCredential = mcp.CallerCredential{Scopes: verified.Credential.Scopes, TaskID: verified.Credential.TaskID}
	}

	tokenSecretResult := auth.NewAccessTokenSecret(cfg.AccessTokenSecret())
	tokenSecret, tokenSecretMatched := tokenSecretResult.(auth.AccessTokenSecretAccepted)
	if !tokenSecretMatched {
		rejected := tokenSecretResult.(auth.AccessTokenSecretRejected)
		logger.Error("load access token secret", "reason", rejected.Reason.Description())
		return 2
	}
	authServiceResult := auth.NewService(db.NewAuthStore(pool), tokenSecret.Value, auth.SystemClock{})
	authService, authServiceMatched := authServiceResult.(auth.ServiceCreated)
	if !authServiceMatched {
		rejected := authServiceResult.(auth.ServiceRejected)
		logger.Error("create auth service", "reason", rejected.Reason.Description())
		return 2
	}

	organizationService := org.NewService(db.NewOrgStore(pool))
	taskStore := db.NewTaskStore(pool)
	taskService := task.NewService(taskStore, organizationService, agentService)
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore, organizationService)
	ledgerService := ledger.NewService(db.NewLedgerStore(pool))
	assetService := assets.NewService(db.NewCollectibleStore(pool))
	notificationService := notification.NewService(db.NewNotificationStore(pool))
	bootstrapAdmins := httpserver.ParseAdminUserIDsForRuntime(os.Getenv("SHARECROP_ADMIN_USER_IDS"))
	platformAdmins := db.NewPlatformAdminStore(pool, bootstrapAdmins)
	moderationTriage := db.NewModerationTriageStore(pool)
	privacyService := db.NewPrivacyStore(pool)
	auditService := audit.NewService(db.NewAuditStore(pool))
	mcpServer := httpserver.NewMCPServer(taskService, submissionService, ledgerService, organizationService, orgCredentialService, assetService, notificationService, authService.Value, platformAdmins, moderationTriage, privacyService, auditService)

	logger.Info("starting sharecrop mcp stdio transport")
	if err := mcp.ServeStdio(ctx, mcpServer, subject, callerCredential, os.Stdin, stdout); err != nil {
		logger.Error("serve mcp stdio", "error", err)
		return 1
	}
	return 0
}

func runServe(ctx context.Context, cfg app.Config, logger *slog.Logger) int {
	staticFiles, err := web.StaticFiles()
	if err != nil {
		logger.Error("load static files", "error", err)
		return 1
	}

	tokenSecretResult := auth.NewAccessTokenSecret(cfg.AccessTokenSecret())
	tokenSecret, tokenSecretMatched := tokenSecretResult.(auth.AccessTokenSecretAccepted)
	if !tokenSecretMatched {
		rejected := tokenSecretResult.(auth.AccessTokenSecretRejected)
		logger.Error("load access token secret", "reason", rejected.Reason.Description())
		return 2
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("open database", "error", err)
		return 1
	}
	defer pool.Close()
	if err := db.VerifyMigrationsCurrent(ctx, pool, cfg.MigrationsDir()); err != nil {
		logger.Error("verify database migrations", "error", err)
		return 1
	}

	authServiceResult := auth.NewService(db.NewAuthStore(pool), tokenSecret.Value, auth.SystemClock{})
	authService, authServiceMatched := authServiceResult.(auth.ServiceCreated)
	if !authServiceMatched {
		rejected := authServiceResult.(auth.ServiceRejected)
		logger.Error("create auth service", "reason", rejected.Reason.Description())
		return 2
	}

	tokenVerifier := auth.NewAccessTokenVerifier(tokenSecret.Value, auth.SystemClock{})
	organizationService := org.NewService(db.NewOrgStore(pool))
	taskStore := db.NewTaskStore(pool)
	agentService := agent.NewService(db.NewAgentStore(pool))
	orgCredentialService := orgcred.NewService(db.NewOrgCredentialStore(pool))
	taskService := task.NewService(taskStore, organizationService, agentService)
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore, organizationService)
	ledgerService := ledger.NewService(db.NewLedgerStore(pool))
	assetService := assets.NewService(db.NewCollectibleStore(pool))
	notificationService := notification.NewService(db.NewNotificationStore(pool))
	bootstrapAdmins := httpserver.ParseAdminUserIDsForRuntime(os.Getenv("SHARECROP_ADMIN_USER_IDS"))

	nativeHandler := httpserver.NewWithRuntimeState(staticFiles, authService.Value, tokenVerifier, organizationService, taskService, submissionService, ledgerService, agentService, orgCredentialService, assetService, httpserver.RuntimeState{
		IPRateLimiter:       db.NewRateLimiter(pool, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec),
		SubjectRateLimiter:  db.NewRateLimiter(pool, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec),
		MCPSessions:         httpserver.NewPersistedMCPHTTPSessionStore(db.NewMCPSessionStore(pool)),
		AuditService:        audit.NewService(db.NewAuditStore(pool)),
		NotificationService: notificationService,
		SavedQueueViews:     db.NewSavedQueueViewStore(pool),
		PrivacyService:      db.NewPrivacyStore(pool),
		PlatformAdmins:      db.NewPlatformAdminStore(pool, bootstrapAdmins),
		ModerationTriage:    db.NewModerationTriageStore(pool),
	})

	// WASI hosting is the default: production runs the same WASM artifact as the
	// browser demo (the app guest embedded via internal/wasiguest), serving the
	// dynamic routes from a pool of reused guest instances whose store calls are
	// dispatched back to Postgres; static assets stay host-side. Set
	// SHARECROP_WASI_MODE=native to run the in-process mux instead;
	// SHARECROP_WASI_GUEST overrides the embedded guest with an external module.
	handler := nativeHandler
	guestWASM, guestSource, err := resolveGuestModule()
	if err != nil {
		logger.Error("resolve wasi guest", "error", err)
		return 1
	}
	switch mode := strings.ToLower(os.Getenv("SHARECROP_WASI_MODE")); {
	case mode == "native":
		logger.Info("serving through the native in-process mux", "reason", "SHARECROP_WASI_MODE=native")
	case len(guestWASM) == 0:
		if mode == "wasi" {
			logger.Error("SHARECROP_WASI_MODE=wasi but no app guest is available; build with `make build` or set SHARECROP_WASI_GUEST")
			return 1
		}
		logger.Warn("no app guest embedded; serving the native in-process mux (build with `make build` for WASI hosting)")
	default:
		guestBuildStart := time.Now()
		wasiHandler, closeGuest, err := serveThroughWASIGuest(ctx, guestWASM, cfg, pool, staticFiles, nativeHandler, shauthConfigured())
		if err != nil {
			logger.Error("build wasi guest host", "error", err)
			return 1
		}
		defer closeGuest()
		// guest_compile is the module-preparation time. It is small (tens of ms)
		// when a baked wazero cache is hit and large (~seconds) when the guest is
		// compiled at startup, so it surfaces whether SHARECROP_WAZERO_CACHE_DIR
		// actually took effect.
		logger.Info("serving dynamic routes through the WASI guest pool",
			"guest", guestSource, "pool_size", wasiPoolSize(),
			"wazero_cache", wazeroCacheSource(), "guest_compile", time.Since(guestBuildStart))
		handler = wasiHandler
	}

	server := &http.Server{
		Addr:              cfg.HTTPAddress(),
		Handler:           handler,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errs := make(chan error, 1)
	go func() {
		logger.Info("starting sharecrop", "address", cfg.HTTPAddress())
		errs <- server.ListenAndServe()
	}()

	stopCtx, stop := signal.NotifyContext(ctx, os.Interrupt, syscall.SIGTERM)
	defer stop()

	select {
	case <-stopCtx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := server.Shutdown(shutdownCtx); err != nil {
			logger.Error("shutdown server", "error", err)
			return 1
		}
		return 0
	case err := <-errs:
		if errors.Is(err, http.ErrServerClosed) {
			return 0
		}
		logger.Error("serve", "error", err)
		return 1
	}
}

// serveThroughWASIGuest builds the handler for the WASI cutover: a pool of the
// compiled app guest (the real mux, over the bridged GuestStores) serves the
// dynamic routes, while static assets and the SPA shell are served host-side
// (the guest carries no static files). Every store call the guest makes is
// dispatched back to the host's Postgres pool via storehost. The returned func
// tears the guest pool down.
func serveThroughWASIGuest(ctx context.Context, guestWASM []byte, cfg app.Config, pool *pgxpool.Pool, staticFiles fs.FS, nativeHandler http.Handler, requireShauthSession bool) (http.Handler, func(), error) {
	// SHARECROP_WAZERO_CACHE_DIR points at a pre-populated wazero compilation
	// cache (baked into the container by the wasi-precompile build step) so the
	// guest's machine code is loaded rather than compiled at startup. Unset falls
	// back to compiling the module on boot.
	cacheDir := os.Getenv("SHARECROP_WAZERO_CACHE_DIR")
	guestPool, err := rpc.NewPoolWithCache(ctx, guestWASM, storehost.Dispatcher(pool), wasiPoolSize(), cacheDir)
	if err != nil {
		return nil, nil, fmt.Errorf("build guest pool: %w", err)
	}
	// The guest runs internal/http's newServer, which reads request-shaping
	// config straight from the environment. Keep this in sync with the guest's
	// local runtime configuration. The Shauth routes remain on the host because
	// the wasip1 guest has no outbound HTTP capability for OpenID Connect
	// discovery and token exchange.
	guestPool.WithGuestEnv(wasiGuestEnvironment(cfg))
	guest := httpbridge.Handler(guestPool)

	mux := http.NewServeMux()
	// Shauth discovery and token exchange use the host network boundary. The
	// remaining application routes continue through the real WASI guest.
	mux.Handle("GET /api/auth/shauth", nativeHandler)
	mux.Handle("GET /api/auth/shauth/callback", nativeHandler)
	// Dynamic routes run in the guest.
	mux.Handle("/api/", guest)
	mux.Handle("/mcp", guest)
	mux.Handle("/healthz", guest)
	// Static assets and the SPA shell are served from the host's embedded files.
	mux.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.FS(staticFiles))))
	mux.Handle("/", applicationShell(staticFiles, requireShauthSession))

	return mux, func() { _ = guestPool.Close(context.Background()) }, nil
}

func shauthConfigured() bool {
	for _, name := range []string{
		"SHARECROP_SHAUTH_ISSUER",
		"SHARECROP_SHAUTH_CLIENT_ID",
		"SHARECROP_SHAUTH_CLIENT_SECRET",
		"SHARECROP_PUBLIC_URL",
	} {
		if os.Getenv(name) == "" {
			return false
		}
	}
	return true
}

func applicationShell(staticFiles fs.FS, requireShauthSession bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if requireShauthSession {
			if _, err := r.Cookie("sharecrop_refresh_token"); err != nil {
				http.Redirect(w, r, "/api/auth/shauth", http.StatusFound)
				return
			}
		}
		data, err := fs.ReadFile(staticFiles, "index.html")
		if err != nil {
			http.Error(w, "index not found", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(data)
	})
}

func wasiGuestEnvironment(cfg app.Config) map[string]string {
	return map[string]string{
		"SHARECROP_ACCESS_TOKEN_SECRET":    cfg.AccessTokenSecret(),
		"SHARECROP_INSECURE_COOKIES":       os.Getenv("SHARECROP_INSECURE_COOKIES"),
		"SHARECROP_ACCOUNT_TOKEN_DELIVERY": os.Getenv("SHARECROP_ACCOUNT_TOKEN_DELIVERY"),
		"SHARECROP_ADMIN_USER_IDS":         os.Getenv("SHARECROP_ADMIN_USER_IDS"),
	}
}

// resolveGuestModule returns the app guest wasm to run: an external module when
// SHARECROP_WASI_GUEST points at one, otherwise the embedded guest (empty when
// the binary was not built for WASI). The second return value labels the source
// for logging.
func resolveGuestModule() ([]byte, string, error) {
	if path := os.Getenv("SHARECROP_WASI_GUEST"); path != "" {
		module, err := os.ReadFile(path)
		if err != nil {
			return nil, "", fmt.Errorf("read guest module %q: %w", path, err)
		}
		return module, path, nil
	}
	return wasiguest.Guest, "embedded", nil
}

// wasiPoolSize is how many guest instances the cutover keeps warm, from
// SHARECROP_WASI_POOL_SIZE (a positive integer) or GOMAXPROCS by default.
func wasiPoolSize() int {
	if raw := os.Getenv("SHARECROP_WASI_POOL_SIZE"); raw != "" {
		if size, err := strconv.Atoi(raw); err == nil && size > 0 {
			return size
		}
	}
	return runtime.GOMAXPROCS(0)
}

// wazeroCacheSource labels where serve looks for a baked wazero compilation
// cache, for the startup log.
func wazeroCacheSource() string {
	if dir := os.Getenv("SHARECROP_WAZERO_CACHE_DIR"); dir != "" {
		return dir
	}
	return "none"
}

// runWASIPrecompile compiles the embedded app guest into a wazero compilation
// cache directory so a later serve loads its machine code instead of compiling
// the module at startup. It is a build-time step (no database or config needed),
// run inside the container build on the same binary and CPU serve will use.
func runWASIPrecompile(ctx context.Context, args []string, logger *slog.Logger) int {
	cacheDir := os.Getenv("SHARECROP_WAZERO_CACHE_DIR")
	if len(args) >= 1 && args[0] != "" {
		cacheDir = args[0]
	}
	if cacheDir == "" {
		logger.Error("wasi-precompile requires a cache directory (argument or SHARECROP_WAZERO_CACHE_DIR)")
		return 2
	}
	guestWASM, source, err := resolveGuestModule()
	if err != nil {
		logger.Error("resolve guest module", "error", err)
		return 1
	}
	if len(guestWASM) == 0 {
		logger.Error("no app guest to precompile; build it with `make wasi-app-guest` first")
		return 1
	}
	start := time.Now()
	if err := rpc.PrecompileGuest(ctx, guestWASM, cacheDir); err != nil {
		logger.Error("precompile guest", "error", err)
		return 1
	}
	logger.Info("precompiled app guest into wazero cache", "guest", source, "cache_dir", cacheDir, "elapsed", time.Since(start))
	return 0
}
