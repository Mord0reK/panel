package main

import (
	"context"
	"log"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"backend/internal/aggregation"
	"backend/internal/api"
	"backend/internal/auth"
	"backend/internal/buffer"
	"backend/internal/config"
	"backend/internal/database"
	ws "backend/internal/websocket"

	"github.com/gorilla/mux"
)

func main() {
	// 1. Load Config
	cfg := config.Load()

	// 2. Setup Logging
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	// 3. Initialize Database
	db, err := database.New(cfg.DatabasePath)
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}
	defer db.Close()

	if err := database.RunMigrations(db); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	// 4. Create Components
	agentHub := ws.NewHub()
	bufferManager := buffer.NewBufferManager()
	aggregator := aggregation.NewAggregator(db)
	bulkInserter := buffer.StartBulkInserter(db, bufferManager)

	// 5. Setup Handlers
	authHandler := api.NewAuthHandler(db, cfg)
	wsHandler := api.NewWebSocketHandler(agentHub, db, bufferManager)
	serversHandler := api.NewServersHandler(db, agentHub)
	commandsHandler := api.NewCommandsHandler(agentHub)
	metricsHandler := api.NewMetricsHandler(db, bufferManager)
	sseHandler := api.NewSSEHandler(db, bufferManager, cfg.CORSOrigin)
	servicesHandler, err := api.NewServicesHandler(db, cfg.JWTSecret)
	if err != nil {
		log.Fatalf("Failed to initialize services handler: %v", err)
	}

	// 6. Router
	r := mux.NewRouter()

	// CORS Middleware
	r.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
			w.Header().Set("Access-Control-Max-Age", "3600")

			if r.Method == "OPTIONS" {
				w.WriteHeader(http.StatusOK)
				return
			}

			next.ServeHTTP(w, r)
		})
	})

	// Handle all OPTIONS requests (preflight)
	r.Methods("OPTIONS").HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Max-Age", "3600")
		w.WriteHeader(http.StatusOK)
	})

	// Public API
	r.HandleFunc("/api/setup", authHandler.HandleSetup).Methods("POST")
	r.HandleFunc("/api/login", authHandler.HandleLogin).Methods("POST")
	r.HandleFunc("/api/auth/status", authHandler.HandleStatus).Methods("GET")

	// Protected API
	authMW := auth.Middleware(cfg.JWTSecret)
	apiRouter := r.PathPrefix("/api").Subrouter()
	apiRouter.Use(authMW)

	apiRouter.HandleFunc("/servers", serversHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/servers/{uuid}", serversHandler.HandleGet).Methods("GET")
	apiRouter.HandleFunc("/servers/{uuid}", serversHandler.HandlePatch).Methods("PATCH")
	apiRouter.HandleFunc("/servers/{uuid}/approve", serversHandler.HandleApprove).Methods("PUT")
	apiRouter.HandleFunc("/servers/{uuid}", serversHandler.HandleDelete).Methods("DELETE")
	apiRouter.HandleFunc("/servers/{uuid}/command", commandsHandler.HandleCommand).Methods("POST")
	apiRouter.HandleFunc("/servers/{uuid}/containers/{id}/command", commandsHandler.HandleContainerCommand).Methods("POST")
	apiRouter.HandleFunc("/servers/{uuid}/containers/{id}/check-update", commandsHandler.HandleCheckUpdate).Methods("POST")
	apiRouter.HandleFunc("/servers/{uuid}/containers/check-updates", commandsHandler.HandleCheckAllUpdates).Methods("POST")
	apiRouter.HandleFunc("/servers/{uuid}/containers/{id}/update", commandsHandler.HandleUpdate).Methods("POST")
	apiRouter.HandleFunc("/servers/{uuid}/containers/{id}", serversHandler.HandleDeleteContainer).Methods("DELETE")
	apiRouter.HandleFunc("/servers/{uuid}/containers", serversHandler.HandleDeleteContainers).Methods("DELETE")

	apiRouter.HandleFunc("/metrics/history/servers/{uuid}", metricsHandler.HandleServerHistory).Methods("GET")
	apiRouter.HandleFunc("/metrics/history/servers/{uuid}/containers/{id}", metricsHandler.HandleContainerHistory).Methods("GET")

	apiRouter.HandleFunc("/metrics/live/all", sseHandler.HandleLiveAll).Methods("GET")
	apiRouter.HandleFunc("/metrics/live/servers/{uuid}", sseHandler.HandleLiveServer).Methods("GET")
	apiRouter.HandleFunc("/services", servicesHandler.HandleList).Methods("GET")
	apiRouter.HandleFunc("/services/{service}/config", servicesHandler.HandleGetConfig).Methods("GET")
	apiRouter.HandleFunc("/services/{service}/config", servicesHandler.HandleConfigUpsert).Methods("PUT")
	apiRouter.HandleFunc("/services/{service}/test", servicesHandler.HandleTestConnection).Methods("POST")
	apiRouter.HandleFunc("/services/{service}/stats", servicesHandler.HandleStats).Methods("GET")

	// WebSocket (Agent)
	r.HandleFunc("/ws/agent", wsHandler.HandleAgent)

	// 7. Start Goroutines
	go agentHub.Run()

	// 8. HTTP Server
	srv := &http.Server{
		Addr:    ":" + cfg.Port,
		Handler: r,
	}

	slog.Info("Server starting", "port", cfg.Port)
	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("listen: %s\n", err)
		}
	}()

	slog.Info("Server started", "port", cfg.Port)

	// 9. Graceful Shutdown
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	slog.Info("Shutting down server...")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := srv.Shutdown(ctx); err != nil {
		log.Fatal("Server forced to shutdown:", err)
	}

	bulkInserter.Stop()
	aggregator.Stop()
	agentHub.Stop()

	// Final flush
	bulkInserter.Flush()

	slog.Info("Server exiting")
}
