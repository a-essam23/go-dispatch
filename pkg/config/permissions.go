package config

import (
	"fmt"
	"sync"

	"github.com/a-essam23/go-dispatch/pkg/state"
)

var (
	registry          = make(map[string]state.Permission)
	nextBit      uint = 2
	registryOnce sync.Once
	mu           sync.RWMutex
)

func init() {
	registryOnce.Do(func() {
		for name, perm := range state.BuiltInPerms {
			registry[name] = perm
		}
	})
}

// GetFullPermissionsBitmap returns a bitmap containing all registered permissions.
func GetFullPermissionsBitmap() state.Permission {
	mu.RLock()
	defer mu.RUnlock()

	var bitmap state.Permission
	for _, p := range registry {
		bitmap |= p
	}
	return bitmap
}

func RegisterPermission(name string) error {
	mu.Lock()
	defer mu.Unlock()

	if _, exists := state.BuiltInPerms[name]; exists {
		return fmt.Errorf("'%s' is reserved for built in permission. please choose a different name", name)
	}

	if _, exists := registry[name]; exists {
		return fmt.Errorf("permission '%s' is already registered", name)
	}
	if nextBit >= 64 {
		return fmt.Errorf("cannot register new permission '%s': maximum of 64 permissions reached", name)
	}

	value := state.Permission(1 << nextBit)
	registry[name] = value
	nextBit++

	return nil
}

// CompilePermissions takes a slice of permission names and returns a combined bitmap.
func CompilePermissions(names []string) (state.Permission, error) {
	mu.RLock()
	defer mu.RUnlock()

	var bitmap state.Permission
	for _, name := range names {
		value, ok := registry[name]
		if !ok {
			return 0, fmt.Errorf("permission '%s' not found", name)
		}
		bitmap |= value
	}
	return bitmap, nil
}

// GetAllRegistered returns a copy of the current permission registry for inspection.
func GetAllRegistered() map[string]state.Permission {
	mu.RLock()
	defer mu.RUnlock()

	regCopy := make(map[string]state.Permission, len(registry))
	for k, v := range registry {
		regCopy[k] = v
	}
	return regCopy
}
