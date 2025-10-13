package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
	jwtlib "github.com/tgym-digital/mini-rproxy/plugins/jwt"
)

var logger *slog.Logger

type LegacyTokenPlugin struct{}

func (p *LegacyTokenPlugin) Init() {
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

func (p *LegacyTokenPlugin) Name() string { return "legacy-token" }

func (p *LegacyTokenPlugin) Handle(ctx *pluginapi.Context) error {
	if ctx.Phase == pluginapi.PhaseRequest {
		maybeEqContext := ctx.Request.Header.Get("x-equipment-context")
		maybeBearer := extractBearer(ctx.Request.Header.Get("authorization"))

		var payload *jwtlib.JWTPayload = nil
		if maybeBearer != "" {
			var err error = nil
			payload, err = jwtlib.ParseJWTPayload(maybeBearer)
			if err != nil {
				logger.Error("invalid jwt structure or payload decode failed", "error", err)
				return nil
			}

			if userToken := jwtlib.BuildLegacyToken(payload); userToken != "" {
				ctx.Request.Header.Set("authorization", "Bearer "+userToken)
			} else {
				logger.Error("got empty user token from jwt", "bearer", maybeBearer)
			}
		}

		if maybeEqContext != "" {
			eqToken, err := jwtlib.EquipmentTokenFromContext(maybeEqContext)
			if err != nil {
				logger.Error(err.Error())
				if payload != nil {
					eqToken, err = jwtlib.EquipmentTokenFromPayload(payload)
					if err != nil {
						logger.Error(err.Error())
					}
				}
			}

			if eqToken != "" {
				logger.Debug("eq token created", "equipment_token", eqToken)
				ctx.Request.Header.Set("x-mwapps-eqtoken", eqToken)
			}
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
