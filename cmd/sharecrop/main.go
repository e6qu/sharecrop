package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

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
	"github.com/e6qu/sharecrop/internal/org"
	"github.com/e6qu/sharecrop/internal/submission"
	"github.com/e6qu/sharecrop/internal/task"
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

	cfgResult := app.LoadConfig()
	cfg, loaded := cfgResult.(app.ConfigLoaded)
	if !loaded {
		rejected := cfgResult.(app.ConfigRejected)
		logger.Error("load config", "reason", rejected.Reason)
		return 2
	}

	if len(args) > 1 {
		switch args[1] {
		case "migrate":
			return runMigrate(ctx, args[2:], cfg.Value, stdout, logger)
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
	if len(args) != 1 || args[0] != "elm-contracts" {
		_, _ = fmt.Fprintln(stdout, "usage: sharecrop generate elm-contracts")
		return 2
	}

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

func runMigrate(ctx context.Context, args []string, cfg app.Config, stdout io.Writer, logger *slog.Logger) int {
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
	secretResult := agent.ParseSecretPlain(rawToken)
	secret, secretMatched := secretResult.(agent.SecretPlainAccepted)
	if !secretMatched {
		logger.Error("agent credential", "reason", "SHARECROP_AGENT_TOKEN is required and must be a valid agent credential")
		return 2
	}

	pool, err := db.Open(ctx, cfg.DatabaseURL())
	if err != nil {
		logger.Error("open database", "error", err)
		return 1
	}
	defer pool.Close()

	agentService := agent.NewService(db.NewAgentStore(pool))
	verifyResult := agentService.Verify(ctx, secret.Value)
	verified, verifiedMatched := verifyResult.(agent.CredentialVerified)
	if !verifiedMatched {
		logger.Error("verify agent credential", "reason", verifyResult.(agent.VerifyRejected).Reason.Description())
		return 1
	}

	organizationService := org.NewService(db.NewOrgStore(pool))
	taskStore := db.NewTaskStore(pool)
	taskService := task.NewService(taskStore, organizationService)
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore, organizationService)
	ledgerService := ledger.NewService(db.NewLedgerStore(pool))
	mcpServer := httpserver.NewMCPServer(taskService, submissionService, ledgerService)

	logger.Info("starting sharecrop mcp stdio transport")
	if err := mcp.ServeStdio(ctx, mcpServer, verified.Subject, verified.Credential.Scopes, os.Stdin, stdout); err != nil {
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
	taskService := task.NewService(taskStore, organizationService)
	submissionService := submission.NewService(db.NewSubmissionStore(pool), taskStore, organizationService)
	ledgerService := ledger.NewService(db.NewLedgerStore(pool))
	agentService := agent.NewService(db.NewAgentStore(pool))
	assetService := assets.NewService(db.NewCollectibleStore(pool))
	notificationService := notification.NewService(db.NewNotificationStore(pool))
	bootstrapAdmins := httpserver.ParseAdminUserIDsForRuntime(os.Getenv("SHARECROP_ADMIN_USER_IDS"))

	server := &http.Server{
		Addr: cfg.HTTPAddress(),
		Handler: httpserver.NewWithRuntimeState(staticFiles, authService.Value, tokenVerifier, organizationService, taskService, submissionService, ledgerService, agentService, assetService, httpserver.RuntimeState{
			IPRateLimiter:       db.NewRateLimiter(pool, "ip", httpserver.IPRateCapacity, httpserver.IPRateRefillPerSec),
			SubjectRateLimiter:  db.NewRateLimiter(pool, "subject", httpserver.MCPRateCapacity, httpserver.MCPRateRefillPerSec),
			MCPSessions:         httpserver.NewPersistedMCPHTTPSessionStore(db.NewMCPSessionStore(pool)),
			AuditService:        audit.NewService(db.NewAuditStore(pool)),
			NotificationService: notificationService,
			SavedQueueViews:     db.NewSavedQueueViewStore(pool),
			PrivacyService:      db.NewPrivacyStore(pool),
			PlatformAdmins:      db.NewPlatformAdminStore(pool, bootstrapAdmins),
			ModerationTriage:    db.NewModerationTriageStore(pool),
		}),
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
