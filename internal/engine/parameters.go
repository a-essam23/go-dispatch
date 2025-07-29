package engine

import (
	"errors"

	"github.com/a-essam23/go-dispatch/pkg/pipeline"
)

type ResolverFunc func(pctx *pipeline.Cargo) (string, error)

// func for param "{$user.id}"
func _userID(pctx *pipeline.Cargo) (string, error) {
	if pctx.User == nil {
		return "", errors.New("param variable 'user.id' is unavailable")
	}
	return pctx.User.ID, nil
}

// func for param "{$conn.id}"
func _connID(pctx *pipeline.Cargo) (string, error) {
	if pctx.Connection == nil {
		return "", errors.New("param variable 'connection.id' is unavailable")
	}
	return pctx.Connection.ID.String(), nil
}

// func for param "{$target.id}"
func _target(pctx *pipeline.Cargo) (string, error) {
	return pctx.TargetID, nil
}
