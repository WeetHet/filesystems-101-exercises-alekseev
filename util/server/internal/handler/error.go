package handler

import (
	"context"
	"fmt"

	"tests-runner/util/logger"
	"tests-runner/util/errors"
	sctx "tests-runner/util/server/context"
)

// LogErrorParams parametrizes LogError
type LogErrorParams struct {
	HTTPMethod string
	HTTPPath   string
	Params     any
	Error      errors.Error
}

// LogError logs handler error adding useful logger fields retrieving
// deriving them from values put into context by ContextBuilder.
func LogError(ctx context.Context, log logger.Logger, p LogErrorParams) {
	fields := logger.Fields{}

	fields[logger.KeyError] = p.Error.Message
	fields["code"] = p.Error.Code
	if p.Error.Reason != nil {
		fields["reason"] = fmt.Sprintf("%+v", p.Error.Reason)
	}
	if p.Error.Details != nil {
		fields["details"] = fmt.Sprintf("%+v", p.Error.Details)
	}
	if p.HTTPMethod != "" {
		fields["http_method"] = p.HTTPMethod
		fields["http_path"] = p.HTTPPath
	}
	if user, err := sctx.User(ctx); err == nil {
		fields["actor_email"] = user.Email
		fields["actor_ext_id"] = user.Federated.UserID
	}
	if acl, err := sctx.ACL(ctx); err == nil {
		fields["actor_id"] = acl.ActorID
	}
	if p.Params != nil {
		fields["params"] = p.Params
	}

	var handlerName string
	if p.HTTPMethod != "" {
		handlerName = "HTTP " + p.HTTPMethod + " " + p.HTTPPath
	}

	logLevel := logger.Error
	if p.Error.IsSilenced() {
		logLevel = logger.Debug
	}
	log.WithFields(fields).Logf(ctx, logLevel, "%s handler error (%s)", handlerName, p.Error.Code)
}
