package server

import (
	"context"
	"errors"
	"log/slog"
	"net"
	"net/http"
	"sync"
	"time"

	"github.com/a-essam23/go-dispatch/internal/router"
	"github.com/a-essam23/go-dispatch/internal/server/middleware"
	"github.com/a-essam23/go-dispatch/pkg/config"
	"github.com/a-essam23/go-dispatch/pkg/state"
	"github.com/a-essam23/go-dispatch/pkg/state/statemanager"
	"github.com/a-essam23/go-dispatch/pkg/transport"
	"github.com/coder/websocket"
	"github.com/google/uuid"
)

type App struct {
	logger       *slog.Logger
	stateManager state.Manager
	eventRouter  *router.EventRouter
	wg           sync.WaitGroup
	http         *http.Server
	config       *config.Config

	ctx context.Context
}

func NewApp(logger *slog.Logger, rootContx context.Context, cfg *config.Config) *App {
	stateManager := statemanager.NewInMemoryManager(logger)
	eventRouter := router.NewEventRouter(logger, stateManager, cfg.Events)

	app := &App{
		logger:       logger,
		stateManager: stateManager,
		eventRouter:  eventRouter,
		config:       cfg,
		ctx:          rootContx,
	}

	mux := http.NewServeMux()
	upgradeHandler := http.HandlerFunc(app.upgradeHandler)
	connCounter := middleware.UserConnectionCounter(stateManager.GetUserConnectionCount)
	// Create a cycler function that closes over the stateManager and logger.
	connCycler := func(userID string) {
		oldest, found := stateManager.FindOldestUserConnection(userID)
		if found {
			logger.Info("Cycling connection: closing oldest", "userID", userID, "connID", oldest.ID)
			oldest.Transport.Close(errors.New("connection cycled by new connection"))
		}
	}

	permCompiler := middleware.PermissionCompiler(config.CompilePermissions)
	mux.Handle("/ws",
		middleware.Chain(upgradeHandler,
			middleware.RequestMetadataMiddleware(),
			middleware.NewRequestLogger(app.logger),
			middleware.NewConnectionLimiter(
				logger,
				connCounter,
				connCycler,
				app.config.Server.ConnectionLimit,
			),
			middleware.NewAuthMiddleware(logger, app.config.Server.Auth.JWTSecret, permCompiler),
		),
	)

	app.http = &http.Server{Addr: app.config.Server.Address, Handler: mux, BaseContext: func(l net.Listener) context.Context {
		return app.ctx
	}}

	return app
}

func (a *App) Run() error {
	go func() {
		a.logger.Info("Server starting", slog.String("addr", a.http.Addr))
		if err := a.http.ListenAndServe(); err != http.ErrServerClosed {
			a.logger.Error("HTTP server failed", slog.Any("error", err))
		}
	}()

	<-a.ctx.Done()
	return a.Shutdown()
}

func (a *App) upgradeHandler(w http.ResponseWriter, r *http.Request) {
	reqMeta, _ := middleware.ReqMetadataFrom(r.Context())
	connLogger := a.logger.With(
		slog.String("remoteAddr", reqMeta.IP),
		slog.String("userID", reqMeta.UserID),
	)

	wsConn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		a.logger.Error("Failed to accept websocket connection", slog.Any("error", err))
		return
	}

	conn := transport.NewConnection(
		r.Context(),
		&a.wg,
		wsConn,
		transport.ConnectionConfig(a.config.Transport),
		nil,
		nil,
		a.logger,
	)
	// register new connection
	stateConn, err := a.stateManager.RegisterConnection(conn, reqMeta.IP)
	if err != nil {
		connLogger.Error("Failed to register connection state", slog.Any("error", err))
		conn.Close(err)
		return
	}
	// associate the authenticated user with the registered connection.
	if _, err := a.stateManager.AssociateUser(stateConn.ID, reqMeta.UserID, reqMeta.GlobalPermissions); err != nil {
		connLogger.Error("Failed to associate user with connection", slog.Any("error", err))
		conn.Close(err)
		return
	}
	conn.SetOnMessageHandler(a.eventRouter.HandleMessage)
	conn.SetOnCloseHandler(func(id uuid.UUID, err error) {
		connLogger.Info("Deregistering connection due to closure", slog.String("connID", id.String()))
		if dErr := a.stateManager.DeregisterConnection(id); dErr != nil {
			connLogger.Error("Failed to deregister connection from state", slog.Any("error", dErr))
		}
	})

	connLogger.Info("User connection fully established", slog.Any("userID", reqMeta.UserID))
	conn.Run()
	<-conn.Done()
}

// graceful shutdown sequence.
func (a *App) Shutdown() error {
	a.logger.Info("Shutting down server...")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := a.http.Shutdown(shutdownCtx); err != nil {
		return err
	}

	// close all active WebSocket connections.
	a.logger.Info("Closing all active connections...")
	allUsers, err := a.stateManager.GetAllUsers()
	if err != nil {
		a.logger.Error(err.Error())
		return err
	}
	for _, user := range allUsers {
		for _, conn := range user.Connections {
			conn.Transport.Close(errors.New("graceful shutdown"))
		}
	}

	// wait for all connection goroutines to finish their cleanup.
	a.wg.Wait()
	a.logger.Info("Server shut down gracefully.")
	return nil
}
