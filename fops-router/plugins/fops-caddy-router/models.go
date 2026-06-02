package fopscaddyrouter

import "time"

const (
	defaultRegistryPath  = "/data/fops-router/registry.json"
	defaultGeneratedPath = "/data/fops-router/generated/routes.Caddyfile"
	defaultCaddyfilePath = "/etc/caddy/Caddyfile"
	tokenEnvName         = "FOPS_ROUTER_API_TOKEN"
	maxRequestBodyBytes  = 1 << 20
)

type Registry struct {
	Stacks map[string]Stack `json:"stacks"`
}

type Stack struct {
	Project   string    `json:"project"`
	Instance  string    `json:"instance"`
	Routes    []Route   `json:"routes"`
	UpdatedAt time.Time `json:"updated_at"`
}

type Route struct {
	ID         string   `json:"id"`
	Hosts      []string `json:"hosts"`
	Entrypoint string   `json:"entrypoint,omitempty"`
	Upstream   Upstream `json:"upstream"`
}

type Upstream struct {
	Scheme                string `json:"scheme"`
	NetworkAlias          string `json:"network_alias"`
	Port                  int    `json:"port"`
	TLSInsecureSkipVerify bool   `json:"tls_insecure_skip_verify,omitempty"`
	TLSServerName         string `json:"tls_server_name,omitempty"`
}

type Status struct {
	OK         bool `json:"ok"`
	StackCount int  `json:"stack_count"`
	RouteCount int  `json:"route_count"`
}

func NewRegistry() Registry {
	return Registry{Stacks: map[string]Stack{}}
}

func stackKey(project, instance string) string {
	return project + "/" + instance
}
