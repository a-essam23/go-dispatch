package router

import (
	"fmt"
	"log/slog"
	"strings"

	"github.com/a-essam23/go-dispatch/pkg/config"
	"github.com/tidwall/gjson"
)

func (r *EventRouter) exectueAction(actx *ActionContext, action config.ActionConfig) error {
	params, err := r.resolveParams(actx, action.Params)
	if err != nil {
		return err
	}
	r.logger.Debug("Executing action", slog.Any("action", action.Name))
	switch action.Name {
	case "_notify_origin":
		return r.actionNotifyOrigin(actx, params)
	default:
		return fmt.Errorf("unknown action '%s", action.Name)
	}
}

func (r *EventRouter) resolveParams(actx *ActionContext, templates []string) ([]string, error) {
	resolved := make([]string, len(templates))
	payloadStr := string(actx.Message.Payload)

	for i, tpl := range templates {
		if !strings.HasPrefix(tpl, "{.") || !strings.HasSuffix(tpl, "}") {
			// just a string not a template
			resolved[i] = tpl
			continue
		}

		// It's a template. Sanitize it by removing the braces.
		path := strings.Trim(tpl, "{.}")
		if path == "payload" {
			// Special case: {.payload} resolves raw payload
			resolved[i] = payloadStr
			continue
		}

		if strings.HasPrefix(path, "payload.") {
			subPath := strings.TrimPrefix(path, "payload.")
			value := gjson.Get(payloadStr, subPath)

			if !value.Exists() {
				resolved[i] = ""
				continue
			}
			resolved[i] = value.String()
			continue
		}

		return nil, fmt.Errorf("unrecognized template path '%s'", path)
	}
	return resolved, nil
}
