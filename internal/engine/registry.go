package engine

import (
	"fmt"
	"log/slog"
	"sync"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

var (
	anRegistry = make(map[string]pipeline.ActionFunc)
	anMu       sync.RWMutex

	modRegistry = make(map[string]pipeline.ModifierFunc)
	modRegMu    sync.RWMutex
)

func RegisterAction(name string, function pipeline.ActionFunc) {
	anMu.Lock()
	defer anMu.Unlock()
	if _, exists := anRegistry[name]; exists {
		panic(fmt.Sprintf("action function already registered: %s", name))
	}
	anRegistry[name] = function
}

// GetActionFunc retrieves an action function from the registry.
func GetActionFunc(name string) (pipeline.ActionFunc, bool) {
	anMu.RLock()
	defer anMu.RUnlock()
	function, ok := anRegistry[name]
	return function, ok
}

func RegisterCoreActions() {
	RegisterAction("_log", actionLog)
	RegisterAction("_notify_origin", actionNotifyOrigin)
	RegisterAction("_notify_room", actionNotifyRoom)
}

// --- Modifier Registry ---

func RegisterModifier(name string, fn pipeline.ModifierFunc) {
	modRegMu.Lock()
	defer modRegMu.Unlock()
	if _, exists := modRegistry[name]; exists {
		panic(fmt.Sprintf("modifier function already registered: %s", name))
	}
	modRegistry[name] = fn
}

func GetModifierFunc(name string) (pipeline.ModifierFunc, bool) {
	modRegMu.RLock()
	defer modRegMu.RUnlock()
	function, ok := modRegistry[name]
	return function, ok
}

func RegisterCoreModifiers(logger *slog.Logger, jwtSecret string) {
	RegisterModifier("secure", newSecureModifier(jwtSecret))
	RegisterModifier("rate_limit", newRateLimitModifier(logger))
}
