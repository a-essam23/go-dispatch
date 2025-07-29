package engine

import (
	"log/slog"
	"sync"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

/*
* The central registry for all executable and context-aware components.
* It is a single, stateful object that holds all registered actions, modifiers, and parameters.
 */
type Registry struct {
	logger   *slog.Logger
	actions  map[string]pipeline.ActionFunc
	actionMu sync.RWMutex

	modifiers  map[string]pipeline.ModifierFunc
	modifierMu sync.RWMutex

	params   map[string]ResolverFunc
	paramsMu sync.RWMutex
}
type RegisterCoreOptions struct {
	JWTsecret string
}

func (e *Registry) RegisterCore(opts *RegisterCoreOptions) {
	e.registerCoreParams()
	e.registerCoreActions()
	e.registerCoreModifiers(opts.JWTsecret)
}

// New creates and initializes a new Engine instance.
func New(logger *slog.Logger) *Registry {
	return &Registry{
		actions:   make(map[string]pipeline.ActionFunc),
		modifiers: make(map[string]pipeline.ModifierFunc),
		params:    make(map[string]ResolverFunc),
		logger:    logger.With(slog.String("component", "engine")),
	}
}

func (e *Registry) registerCoreActions() {
	e.RegisterAction("_log", actionLog)
	e.RegisterAction("_join", actionJoinRoom)
	e.RegisterAction("_leave", actionLeaveRoom)

	e.RegisterAction("_notify_origin", actionNotifyOrigin)
	e.RegisterAction("_notify_room", actionNotifyRoom)
	e.logger.Info("Resgisted core actions", slog.Any("count", len(e.actions)))
}

func (e *Registry) registerCoreModifiers(jwtSecret string) {
	e.RegisterModifier("secure", newSecureModifier(jwtSecret))
	e.RegisterModifier("rate_limit", newRateLimitModifier(e.logger))
	e.logger.Info("Resgisted core modifiers", slog.Any("count", len(e.modifiers)))
}

func (e *Registry) registerCoreParams() {
	e.RegisterParams("target.id", _target)
	e.RegisterParams("conn.id", _connID)
	e.RegisterParams("user.id", _userID)
	e.logger.Info("Resgisted core params", slog.Any("count", len(e.params)))
}

// --- Action Methods ---
func (e *Registry) RegisterAction(name string, fn pipeline.ActionFunc) {
	e.actionMu.Lock()
	defer e.actionMu.Unlock()
	if _, exists := e.actions[name]; exists {
		panic("action function already registered: " + name)
	}
	e.actions[name] = fn
}

func (e *Registry) GetActionFunc(name string) (pipeline.ActionFunc, bool) {
	e.actionMu.RLock()
	defer e.actionMu.RUnlock()
	fn, ok := e.actions[name]
	return fn, ok
}

// --- Modifier Methods ---

func (e *Registry) RegisterModifier(name string, fn pipeline.ModifierFunc) {
	e.modifierMu.Lock()
	defer e.modifierMu.Unlock()
	if _, exists := e.modifiers[name]; exists {
		panic("modifier function already registered: " + name)
	}
	e.modifiers[name] = fn
}

func (e *Registry) GetModifierFunc(name string) (pipeline.ModifierFunc, bool) {
	e.modifierMu.RLock()
	defer e.modifierMu.RUnlock()
	fn, ok := e.modifiers[name]
	return fn, ok
}

// --- Params Methods ---

func (e *Registry) RegisterParams(name string, resolver ResolverFunc) {
	e.paramsMu.Lock()
	defer e.paramsMu.Unlock()
	if _, exists := e.params[name]; exists {
		panic("Param already registered: " + name)
	}
	e.params[name] = resolver
}

func (e *Registry) GetParamResolver(name string) (ResolverFunc, bool) {
	e.paramsMu.RLock()
	defer e.paramsMu.RUnlock()
	resolver, ok := e.params[name]
	return resolver, ok
}

// GetAllRegisteredParams returns all registered variable names for validation.
func (e *Registry) GetAllRegisteredParams() []string {
	e.paramsMu.RLock()
	defer e.paramsMu.RUnlock()
	keys := make([]string, 0, len(e.params))
	for k := range e.params {
		keys = append(keys, k)
	}
	return keys
}
