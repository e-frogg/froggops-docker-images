package fopscaddyrouter

import (
	"fmt"
	"os"

	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/caddyconfig"
	_ "github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

func NewCaddyReloader(caddyfilePath string) ConfigReloader {
	if caddyfilePath == "" {
		caddyfilePath = defaultCaddyfilePath
	}

	return func() error {
		body, err := os.ReadFile(caddyfilePath)
		if err != nil {
			return err
		}

		adapter := caddyconfig.GetAdapter("caddyfile")
		if adapter == nil {
			return fmt.Errorf("caddyfile adapter is not registered")
		}

		config, warnings, err := adapter.Adapt(body, map[string]any{"filename": caddyfilePath})
		if err != nil {
			return err
		}
		for _, warning := range warnings {
			caddy.Log().Named("admin.api.fops_router").Warn(warning.String())
		}

		return caddy.Load(config, true)
	}
}
