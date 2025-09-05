// go build -buildmode=plugin -o plugins/headerdemo/headerdemo.so ./plugins/headerdemo
package main

import (
	"github.com/tgym-digital/mini-rproxy/core/pluginapi"
)

type HeaderDemo struct{}

func (h *HeaderDemo) Name() string { return "header-demo" }

func (h *HeaderDemo) Handle(ctx *pluginapi.Context) error {
	switch ctx.Phase {
	case pluginapi.PhaseRequest:
		ctx.Request.Header.Set("X-Demo-Request", "sbrobris")
	case pluginapi.PhaseResponse:
		if ctx.Response != nil {
			ctx.Response.Header.Set("X-Demo-Response", "sbrobris")
		}
	}
	return nil
}

var MiniRProxyPluginInstance pluginapi.Plugin = &HeaderDemo{}
