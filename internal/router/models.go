package router

import "encoding/json"

type ClientMessage struct {
	Target  string          `json:"target"`
	Event   string          `json:"event"`
	Payload json.RawMessage `json:"payload"`
}
