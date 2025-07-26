package router

import (
	"encoding/json"
	"errors"
	"fmt"
)

type notifyOriginParams struct {
	EventName string
	Payload   string
}

func (r *EventRouter) actionNotifyOrigin(actx *ActionContext, params []string) error {
	if len(params) != 2 {
		return errors.New("_notify_origin requires exactly 2 parameters:[eventName, paylpad]")
	}
	parsed := notifyOriginParams{
		EventName: params[0],
		Payload:   params[1],
	}
	responsePayload := json.RawMessage(parsed.Payload)
	response := ClientMessage{
		Event:   parsed.EventName,
		Payload: responsePayload,
	}
	msgBytes, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response for _notify_origin: %w", err)
	}
	actx.Conn.Transport.Send(msgBytes)
	return nil
}
