package config

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/a-essam23/go-dispatch/internal/engine"
	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type ActionFuncProvider func(name string) (pipeline.ActionFunc, bool)
type ModifierFuncProvider func(name string) (pipeline.ModifierFunc, bool)

func CompilePipelines(cfg *Config, e *engine.Registry) error {
	cfg.Pipelines = make(map[string]*pipeline.CompiledPipeline)

	for eventName, eventCfg := range cfg.Events {
		compiledPipe := &pipeline.CompiledPipeline{
			Modifiers: make([]pipeline.ModifierStep, 0, len(eventCfg.Modifiers)),
			Actions:   make([]pipeline.Step, 0, len(eventCfg.Actions)),
		}

		// Compile Modifiers
		for _, modCfg := range eventCfg.Modifiers {
			fn, ok := e.GetModifierFunc(modCfg.Name)
			if !ok {
				return fmt.Errorf("unknown modifier '%s' in event '%s'", modCfg.Name, eventName)
			}
			if err := validateParams(modCfg.Params, e); err != nil {
				return fmt.Errorf("invalid params for modifier '%s' in event '%s': %w", modCfg.Name, eventName, err)
			}
			step := pipeline.ModifierStep{
				Function: fn,
				Params:   modCfg.Params,
			}
			compiledPipe.Modifiers = append(compiledPipe.Modifiers, step)
		}

		for _, actionCfg := range eventCfg.Actions {
			fn, ok := e.GetActionFunc(actionCfg.Name)
			if !ok {
				return fmt.Errorf("unknown action '%s' in event '%s'", actionCfg.Name, eventName)
			}
			if err := validateParams(actionCfg.Params, e); err != nil {
				return fmt.Errorf("invalid params for action '%s' in event '%s': %w", actionCfg.Name, eventName, err)
			}
			step := pipeline.Step{
				Function: fn,
				Params:   actionCfg.Params,
			}
			compiledPipe.Actions = append(compiledPipe.Actions, step)
		}

		cfg.Pipelines[eventName] = compiledPipe
	}
	cfg.Events = nil
	return nil
}

var contextVarRegex = regexp.MustCompile(`{\$([a-zA-Z0-9_.-]+)}`)

func validateParams(params []string, e *engine.Registry) error {
	for _, p := range params {
		matches := contextVarRegex.FindAllStringSubmatch(p, -1)
		for _, match := range matches {
			if len(match) < 2 {
				continue // Should not happen with a correct regex
			}
			varName := match[1] // The captured group, e.g., "user.id"

			// Check against registered context variables, including dynamic token claims.
			if strings.HasPrefix(varName, "token.") {
				// We don't validate specific token claims at compile time,
				// as they are dynamic, but we know 'token.' is a valid prefix.
				continue
			}

			if _, ok := e.GetParamResolver(varName); !ok {
				return fmt.Errorf("invalid context variable '{$%s}'", varName)
			}
		}
	}
	return nil
}
