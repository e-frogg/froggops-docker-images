package fopscaddyrouter

import (
	"bytes"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/caddyserver/caddy/v2"
)

func TestParseStackPath(t *testing.T) {
	project, instance, err := parseStackPath("/fops-router/v1/stacks/example-app/production")
	if err != nil {
		t.Fatalf("parseStackPath() error = %v", err)
	}
	if project != "example-app" || instance != "production" {
		t.Fatalf("unexpected project/instance: %s/%s", project, instance)
	}
}

func TestParseStackPathDecodesSegments(t *testing.T) {
	project, instance, err := parseStackPath("/fops-router/v1/stacks/example-app/production%2Dblue")
	if err != nil {
		t.Fatalf("parseStackPath() error = %v", err)
	}
	if project != "example-app" || instance != "production-blue" {
		t.Fatalf("unexpected project/instance: %s/%s", project, instance)
	}
}

func TestDecodeJSONBodyRejectsUnknownFields(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/fops-router/v1/stacks/example-app/production", strings.NewReader(`{"project":"example-app","instance":"production","routes":[],"unexpected":true}`))
	recorder := httptest.NewRecorder()

	var stack Stack
	if err := decodeJSONBody(recorder, req, &stack); err == nil {
		t.Fatal("expected unknown field error")
	}
}

func TestDecodeJSONBodyRejectsMultipleDocuments(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/fops-router/v1/stacks/example-app/production", strings.NewReader(`{"project":"example-app","instance":"production","routes":[]} {}`))
	recorder := httptest.NewRecorder()

	var stack Stack
	if err := decodeJSONBody(recorder, req, &stack); err == nil {
		t.Fatal("expected multiple document error")
	}
}

func TestDecodeJSONBodyRejectsTooLargeBody(t *testing.T) {
	req := httptest.NewRequest(http.MethodPut, "/fops-router/v1/stacks/example-app/production", bytes.NewReader(bytes.Repeat([]byte("x"), maxRequestBodyBytes+1)))
	recorder := httptest.NewRecorder()

	var stack Stack
	if err := decodeJSONBody(recorder, req, &stack); err == nil {
		t.Fatal("expected too large body error")
	}
}

func TestHandleStackReturnsBadRequestForValidationError(t *testing.T) {
	dir := t.TempDir()
	handler := &AdminHandler{
		manager: NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "routes.Caddyfile"), nil),
	}

	stack := sampleStack("example-app", "production", "web")
	stack.Routes[0].Hosts = []string{"invalid-host"}
	body := fmt.Sprintf(`{
		"project": %q,
		"instance": %q,
		"routes": [{
			"id": %q,
			"hosts": [%q],
			"entrypoint": %q,
			"upstream": {
				"scheme": %q,
				"network_alias": %q,
				"port": %d
			}
		}]
	}`, stack.Project, stack.Instance, stack.Routes[0].ID, stack.Routes[0].Hosts[0], stack.Routes[0].Entrypoint, stack.Routes[0].Upstream.Scheme, stack.Routes[0].Upstream.NetworkAlias, stack.Routes[0].Upstream.Port)

	req := httptest.NewRequest(http.MethodPut, "/fops-router/v1/stacks/example-app/production", strings.NewReader(body))
	err := handler.handleStack(httptest.NewRecorder(), req)

	apiErr := assertAPIError(t, err, http.StatusBadRequest)
	if !strings.Contains(apiErr.Error(), "must be a DNS name") {
		t.Fatalf("expected validation detail, got %q", apiErr.Error())
	}
}

func TestHandleStackReturnsGenericInternalErrorWhenReloadFails(t *testing.T) {
	dir := t.TempDir()
	handler := &AdminHandler{
		manager: NewManager(filepath.Join(dir, "registry.json"), filepath.Join(dir, "routes.Caddyfile"), func() error {
			return errors.New("reload failed: /etc/caddy/Caddyfile: secret detail")
		}),
	}

	body := `{
		"project": "example-app",
		"instance": "production",
		"routes": [{
			"id": "web",
			"hosts": ["app.example.com"],
			"entrypoint": "example-app-production-web",
			"upstream": {
				"scheme": "https",
				"network_alias": "example-app-caddy",
				"port": 443
			}
		}]
	}`
	req := httptest.NewRequest(http.MethodPut, "/fops-router/v1/stacks/example-app/production", strings.NewReader(body))
	err := handler.handleStack(httptest.NewRecorder(), req)

	apiErr := assertAPIError(t, err, http.StatusInternalServerError)
	if apiErr.Error() != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", apiErr.Error())
	}
	if strings.Contains(apiErr.Error(), "secret detail") {
		t.Fatalf("internal detail leaked to client: %q", apiErr.Error())
	}
}

func TestHandleStatusReturnsGenericInternalError(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(registryPath, []byte("{"), 0644); err != nil {
		t.Fatalf("failed to write invalid registry: %v", err)
	}
	handler := &AdminHandler{
		manager: NewManager(registryPath, filepath.Join(dir, "routes.Caddyfile"), nil),
	}

	req := httptest.NewRequest(http.MethodGet, "/fops-router/v1/status", nil)
	err := handler.handleStatus(httptest.NewRecorder(), req)

	apiErr := assertAPIError(t, err, http.StatusInternalServerError)
	if apiErr.Error() != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", apiErr.Error())
	}
	if strings.Contains(apiErr.Error(), "invalid character") {
		t.Fatalf("internal detail leaked to client: %q", apiErr.Error())
	}
}

func TestHandleStacksReturnsGenericInternalError(t *testing.T) {
	dir := t.TempDir()
	registryPath := filepath.Join(dir, "registry.json")
	if err := os.WriteFile(registryPath, []byte("{"), 0644); err != nil {
		t.Fatalf("failed to write invalid registry: %v", err)
	}
	handler := &AdminHandler{
		manager: NewManager(registryPath, filepath.Join(dir, "routes.Caddyfile"), nil),
	}

	req := httptest.NewRequest(http.MethodGet, "/fops-router/v1/stacks", nil)
	err := handler.handleStacks(httptest.NewRecorder(), req)

	apiErr := assertAPIError(t, err, http.StatusInternalServerError)
	if apiErr.Error() != "internal server error" {
		t.Fatalf("expected generic internal error, got %q", apiErr.Error())
	}
	if strings.Contains(apiErr.Error(), "invalid character") {
		t.Fatalf("internal detail leaked to client: %q", apiErr.Error())
	}
}

func assertAPIError(t *testing.T, err error, status int) caddy.APIError {
	t.Helper()

	var apiErr caddy.APIError
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected caddy.APIError, got %T: %v", err, err)
	}
	if apiErr.HTTPStatus != status {
		t.Fatalf("expected HTTP status %d, got %d (%v)", status, apiErr.HTTPStatus, apiErr)
	}
	if apiErr.Err == nil && apiErr.Message == "" {
		t.Fatal("expected API error message")
	}
	return apiErr
}
