package config

import (
	"fmt"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type ActionFuncProvider func(name string) (pipeline.ActionFunc, bool)

func CompilePipelines(cfg *Config, provider ActionFuncProvider) error {
	cfg.Pipelines = make(map[string][]pipeline.Step)
	for eventName, eventCfg := range cfg.Events {
		pipe := make([]pipeline.Step, 0, len(eventCfg.Actions))
		for _, actionCfg := range eventCfg.Actions {
			// look up the Go function for this action name.
			fn, ok := provider(actionCfg.Name)
			if !ok {
				return fmt.Errorf("unkown action '%s' in event '%s'", actionCfg.Name, eventName)
			}
			// create the executable step and add it to pipeline
			step := pipeline.Step{
				Function: fn,
				Params:   actionCfg.Params,
			}
			pipe = append(pipe, step)
		}
		cfg.Pipelines[eventName] = pipe
	}
	cfg.Events = nil
	return nil
}
