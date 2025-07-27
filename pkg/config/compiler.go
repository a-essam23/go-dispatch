package config

import (
	"fmt"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type ActionFuncProvider func(name string) (pipeline.ActionFunc, bool)
type ModifierFuncProvider func(name string) (pipeline.ModifierFunc, bool)

func CompilePipelines(cfg *Config, anProvider ActionFuncProvider, modProvider ModifierFuncProvider) error {
	cfg.Pipelines = make(map[string]*pipeline.CompiledPipeline)

	for eventName, eventCfg := range cfg.Events {
		compiledPipe := &pipeline.CompiledPipeline{
			Modifiers: make([]pipeline.ModifierStep, 0, len(eventCfg.Modifiers)),
			Actions:   make([]pipeline.Step, 0, len(eventCfg.Actions)),
		}

		// Compile Modifiers
		if modProvider != nil {
			for _, modCfg := range eventCfg.Modifiers {
				fn, ok := modProvider(modCfg.Name)
				if !ok {
					return fmt.Errorf("unknown modifier '%s' in event '%s'", modCfg.Name, eventName)
				}
				step := pipeline.ModifierStep{
					Function: fn,
					Params:   modCfg.Params,
				}
				compiledPipe.Modifiers = append(compiledPipe.Modifiers, step)
			}
		}
		if anProvider != nil {
			for _, actionCfg := range eventCfg.Actions {
				// look up the Go function for this action name.
				fn, ok := anProvider(actionCfg.Name)
				if !ok {
					return fmt.Errorf("unkown action '%s' in event '%s'", actionCfg.Name, eventName)
				}
				// create the executable step and add it to pipeline
				step := pipeline.Step{
					Function: fn,
					Params:   actionCfg.Params,
				}
				compiledPipe.Actions = append(compiledPipe.Actions, step)
			}
		}
		cfg.Pipelines[eventName] = compiledPipe
	}
	cfg.Events = nil
	return nil
}
