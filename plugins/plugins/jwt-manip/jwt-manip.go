package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
	jwtlib "github.com/tgym-digital/mini-rproxy/plugins/jwt"
)

var logger *slog.Logger

func init() {
	mode := strings.ToLower(os.Getenv("LOG_MODE"))
	var handler slog.Handler
	if mode == "json" {
		handler = slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	} else {
		handler = slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})
	}
	logger = slog.New(handler)
	logger.Info("legacy-token plugin initialized", "log_mode", mode)
}

type LegacyTokenPlugin struct{}

func (p *LegacyTokenPlugin) Name() string { return "legacy-token" }

func (p *LegacyTokenPlugin) Handle(ctx *pluginapi.Context) error {
	if ctx.Phase == pluginapi.PhaseRequest {
		maybeEqContext := ctx.Request.Header.Get("X-Equipment-Context")
		maybeBearer := extractBearer(ctx.Request.Header.Get("Authorization"))
		if maybeBearer == "" {
			logger.Debug("authorization header not bearer")
			return nil
		}

		payload, err := jwtlib.ParseJWTPayload(maybeBearer)
		if err != nil {
			logger.Error("invalid jwt structure or payload decode failed", "error", err)
			return nil
		}

		userToken := jwtlib.BuildLegacyToken(payload)
		eqToken := jwtlib.BuildEquipmentToken(payload, maybeEqContext)
		if eqToken != "" {
			ctx.Request.Header.Set("x-mwapps-eqtoken", eqToken)
		}
		if userToken != "" {
			ctx.Request.Header.Set("authorization", userToken)
		}
	}
	return nil
}

func extractBearer(h string) string {
	if !strings.HasPrefix(strings.ToLower(h), "bearer ") {
		return ""
	}
	return strings.TrimSpace(h[7:])
}

var MiniRProxyPluginInstance pluginapi.Plugin = &LegacyTokenPlugin{}
