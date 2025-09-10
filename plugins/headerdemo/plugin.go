package main

import (
	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
)

type HeaderDemo struct{}

func (h *HeaderDemo) Name() string { return "header-demo" }

func (h *HeaderDemo) Handle(ctx *pluginapi.Context) error {
	switch ctx.Phase {
	case pluginapi.PhaseRequest:
		ctx.Request.Header.Set("x-demo-request", "hello")
	case pluginapi.PhaseResponse:
		if ctx.Response != nil {
			ctx.Response.Header.Set("x-demo-response", "hello")
		}
	}
	return nil
}

var MiniRProxyPluginInstance pluginapi.Plugin = &HeaderDemo{}
