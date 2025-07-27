package engine

import (
	"fmt"
	"sync"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

var (
	registry = make(map[string]pipeline.ActionFunc)
	regMu    sync.RWMutex
)

func RegisterAction(name string, function pipeline.ActionFunc) {
	regMu.Lock()
	defer regMu.Unlock()
	if _, exists := registry[name]; exists {
		panic(fmt.Sprintf("action function already registered: %s", name))
	}
	registry[name] = function
}

// GetActionFunc retrieves an action function from the registry.
func GetActionFunc(name string) (pipeline.ActionFunc, bool) {
	regMu.RLock()
	defer regMu.RUnlock()
	function, ok := registry[name]
	return function, ok
}

func RegisterCoreActions() {
	RegisterAction("_log", actionLog)
	RegisterAction("_notify_origin", actionNotifyOrigin)
	RegisterAction("_notify_room", actionNotifyRoom)
}
