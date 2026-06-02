package fopscaddyrouter

import (
	"fmt"
	"net"
	"regexp"
	"strings"
)

var safeTokenPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)
var dnsLabelPattern = regexp.MustCompile(`^[A-Za-z0-9](?:[A-Za-z0-9-]{0,61}[A-Za-z0-9])?$`)

func validateRegistry(registry Registry) error {
	owners := map[string]string{}
	for key, stack := range registry.Stacks {
		if err := validateStack(key, stack); err != nil {
			return err
		}
		for _, route := range stack.Routes {
			for _, host := range route.Hosts {
				normalizedHost := strings.ToLower(strings.TrimSpace(host))
				if owner, exists := owners[normalizedHost]; exists && owner != key {
					return validationErrorf("host %q is already owned by stack %q", host, owner)
				}
				owners[normalizedHost] = key
			}
		}
	}

	return nil
}

func validateStack(expectedKey string, stack Stack) error {
	if err := validateStackIdentity(stack.Project, stack.Instance); err != nil {
		return err
	}
	if expectedKey != stackKey(stack.Project, stack.Instance) {
		return validationErrorf("stack key %q does not match payload %q", expectedKey, stackKey(stack.Project, stack.Instance))
	}

	routeIDs := map[string]bool{}
	hosts := map[string]bool{}
	for _, route := range stack.Routes {
		if err := validateRoute(route); err != nil {
			return fmt.Errorf("route %q: %w", route.ID, err)
		}
		if routeIDs[route.ID] {
			return validationErrorf("duplicate route id %q", route.ID)
		}
		routeIDs[route.ID] = true

		for _, host := range route.Hosts {
			normalizedHost := strings.ToLower(strings.TrimSpace(host))
			if hosts[normalizedHost] {
				return validationErrorf("duplicate host %q in stack %q", host, expectedKey)
			}
			hosts[normalizedHost] = true
		}
	}

	return nil
}

func validateRoute(route Route) error {
	if !safeTokenPattern.MatchString(route.ID) {
		return validationErrorf("invalid id %q", route.ID)
	}
	if route.Entrypoint != "" && !safeTokenPattern.MatchString(route.Entrypoint) {
		return validationErrorf("invalid entrypoint %q", route.Entrypoint)
	}
	if len(route.Hosts) == 0 {
		return validationErrorf("at least one host is required")
	}
	for _, host := range route.Hosts {
		if err := validateHost(host); err != nil {
			return err
		}
	}
	if route.Upstream.Scheme != "http" && route.Upstream.Scheme != "https" {
		return validationErrorf("invalid upstream scheme %q", route.Upstream.Scheme)
	}
	if !safeTokenPattern.MatchString(route.Upstream.NetworkAlias) {
		return validationErrorf("invalid upstream network_alias %q", route.Upstream.NetworkAlias)
	}
	if route.Upstream.Port < 1 || route.Upstream.Port > 65535 {
		return validationErrorf("invalid upstream port %d", route.Upstream.Port)
	}
	if route.Upstream.TLSServerName != "" {
		if err := validateHost(route.Upstream.TLSServerName); err != nil {
			return fmt.Errorf("invalid upstream tls_server_name: %w", err)
		}
	}

	return nil
}

func validateStackIdentity(project string, instance string) error {
	if !safeTokenPattern.MatchString(project) {
		return validationErrorf("invalid project %q", project)
	}
	if !safeTokenPattern.MatchString(instance) {
		return validationErrorf("invalid instance %q", instance)
	}

	return nil
}

func validateHost(host string) error {
	if host != strings.TrimSpace(host) {
		return validationErrorf("host %q must not have surrounding whitespace", host)
	}
	if host == "" {
		return validationErrorf("empty host")
	}
	if strings.ContainsAny(host, " \t\r\n{},\"'/#\\") {
		return validationErrorf("invalid host %q", host)
	}
	if strings.HasPrefix(host, "*.") {
		host = strings.TrimPrefix(host, "*.")
	}
	if net.ParseIP(host) != nil {
		if strings.Contains(host, ":") {
			return validationErrorf("IPv6 host %q is not supported", host)
		}
		return nil
	}
	if strings.Contains(host, ":") {
		return validationErrorf("host %q must not include a port", host)
	}
	labels := strings.Split(host, ".")
	if len(labels) < 2 {
		return validationErrorf("host %q must be a DNS name with at least two labels", host)
	}
	for _, label := range labels {
		if !dnsLabelPattern.MatchString(label) {
			return validationErrorf("invalid host %q", host)
		}
	}

	return nil
}
