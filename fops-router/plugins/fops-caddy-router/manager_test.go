package fopscaddyrouter

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestManagerUpsertGeneratesAndPersistsRoute(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "generated", "routes.Caddyfile"), nil)

	stack := sampleStack("example-app", "production", "web")
	if _, err := manager.UpsertStack("example-app", "production", stack); err != nil {
		t.Fatalf("UpsertStack() error = %v", err)
	}

	generated, err := os.ReadFile(filepath.Join(dir, "generated", "routes.Caddyfile"))
	if err != nil {
		t.Fatalf("generated file missing: %v", err)
	}

	content := string(generated)
	assertContains(t, content, "app.example.com {")
	assertContains(t, content, "import /etc/fops-router/entrypoint.Caddyfile example-app-production-web")
	assertContains(t, content, "reverse_proxy https://example-app-caddy:443")
	assertContains(t, content, "tls_server_name app.example.com")
	assertContains(t, content, "tls_insecure_skip_verify")
}

func TestManagerRejectsHostCollision(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "routes.Caddyfile"), nil)

	if _, err := manager.UpsertStack("example-app", "production", sampleStack("example-app", "production", "web")); err != nil {
		t.Fatalf("first UpsertStack() error = %v", err)
	}

	colliding := sampleStack("other-app", "production", "main")
	colliding.Routes[0].Hosts = []string{"app.example.com"}
	if _, err := manager.UpsertStack("other-app", "production", colliding); err == nil {
		t.Fatal("expected host collision error")
	} else {
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T: %v", err, err)
		}
	}
}

func TestManagerRollsBackFilesWhenReloadFails(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "routes.Caddyfile"), nil)

	if _, err := manager.UpsertStack("example-app", "production", sampleStack("example-app", "production", "web")); err != nil {
		t.Fatalf("first UpsertStack() error = %v", err)
	}

	manager.reloader = func() error {
		return errors.New("reload failed")
	}
	if _, err := manager.UpsertStack("other-app", "production", sampleStack("other-app", "production", "main")); err == nil {
		t.Fatal("expected reload error")
	}

	registry, err := manager.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if len(registry.Stacks) != 1 {
		t.Fatalf("expected rollback to one stack, got %d", len(registry.Stacks))
	}

	generated, err := os.ReadFile(filepath.Join(dir, "routes.Caddyfile"))
	if err != nil {
		t.Fatalf("generated file missing: %v", err)
	}
	if strings.Contains(string(generated), "other-app") {
		t.Fatalf("generated file was not rolled back:\n%s", generated)
	}
}

func TestManagerDeleteStack(t *testing.T) {
	dir := t.TempDir()
	manager := NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "routes.Caddyfile"), nil)

	if _, err := manager.UpsertStack("example-app", "production", sampleStack("example-app", "production", "web")); err != nil {
		t.Fatalf("UpsertStack() error = %v", err)
	}
	if _, err := manager.DeleteStack("example-app", "production"); err != nil {
		t.Fatalf("DeleteStack() error = %v", err)
	}

	registry, err := manager.LoadRegistry()
	if err != nil {
		t.Fatalf("LoadRegistry() error = %v", err)
	}
	if len(registry.Stacks) != 0 {
		t.Fatalf("expected empty registry, got %d stacks", len(registry.Stacks))
	}
}

func TestManagerRejectsInvalidHostBeforeWriting(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.json")
	generatedPath := filepath.Join(dir, "routes.Caddyfile")
	manager := NewManager(registryPath, generatedPath, nil)

	stack := sampleStack("example-app", "production", "web")
	stack.Routes[0].Hosts = []string{"app.example.com,admin.example.com"}
	if _, err := manager.UpsertStack("example-app", "production", stack); err == nil {
		t.Fatal("expected invalid host error")
	} else {
		var validationErr *ValidationError
		if !errors.As(err, &validationErr) {
			t.Fatalf("expected ValidationError, got %T: %v", err, err)
		}
	}

	if _, err := os.Stat(registryPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("registry should not be written, stat error = %v", err)
	}
	if _, err := os.Stat(generatedPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("generated file should not be written, stat error = %v", err)
	}
}

func sampleStack(project string, instance string, routeID string) Stack {
	return Stack{
		Project:  project,
		Instance: instance,
		Routes: []Route{
			{
				ID:         routeID,
				Hosts:      []string{"app.example.com"},
				Entrypoint: project + "-" + instance + "-" + routeID,
				Upstream: Upstream{
					Scheme:                "https",
					NetworkAlias:          project + "-caddy",
					Port:                  443,
					TLSInsecureSkipVerify: true,
				},
			},
		},
	}
}

func assertContains(t *testing.T, content string, expected string) {
	t.Helper()
	if !strings.Contains(content, expected) {
		t.Fatalf("expected generated content to contain %q:\n%s", expected, content)
	}
}
