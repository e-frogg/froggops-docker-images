package fopscaddyrouter

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/caddyserver/caddy/v2"
	"go.uber.org/zap"
)

func init() {
	caddy.RegisterModule(AdminHandler{})
}

type AdminHandler struct {
	manager *Manager
}

func (AdminHandler) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "admin.api.fops_router",
		New: func() caddy.Module { return new(AdminHandler) },
	}
}

func (h *AdminHandler) Routes() []caddy.AdminRoute {
	h.manager = NewManager(defaultRegistryPath, defaultGeneratedPath, NewCaddyReloader(defaultCaddyfilePath))

	return []caddy.AdminRoute{
		{Pattern: "/fops-router/v1/status", Handler: caddy.AdminHandlerFunc(h.handleStatus)},
		{Pattern: "/fops-router/v1/stacks", Handler: caddy.AdminHandlerFunc(h.handleStacks)},
		{Pattern: "/fops-router/v1/stacks/", Handler: caddy.AdminHandlerFunc(h.handleStack)},
	}
}

func (h *AdminHandler) handleStatus(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return apiError(http.StatusMethodNotAllowed, "method not allowed")
	}

	status, err := h.manager.Status()
	if err != nil {
		return internalAPIError(r, err)
	}

	return writeJSON(w, status)
}

func (h *AdminHandler) handleStacks(w http.ResponseWriter, r *http.Request) error {
	if r.Method != http.MethodGet {
		return apiError(http.StatusMethodNotAllowed, "method not allowed")
	}

	registry, err := h.manager.LoadRegistry()
	if err != nil {
		return internalAPIError(r, err)
	}

	return writeJSON(w, registry)
}

func (h *AdminHandler) handleStack(w http.ResponseWriter, r *http.Request) error {
	project, instance, err := parseStackPath(r.URL.Path)
	if err != nil {
		return apiError(http.StatusNotFound, err.Error())
	}

	switch r.Method {
	case http.MethodPut:
		var stack Stack
		if err := decodeJSONBody(w, r, &stack); err != nil {
			return apiError(http.StatusBadRequest, err.Error())
		}
		registry, err := h.manager.UpsertStack(project, instance, stack)
		if err != nil {
			return mutationAPIError(r, err)
		}

		return writeJSON(w, registry.Stacks[stackKey(project, instance)])
	case http.MethodDelete:
		registry, err := h.manager.DeleteStack(project, instance)
		if err != nil {
			return mutationAPIError(r, err)
		}

		return writeJSON(w, registry)
	default:
		return apiError(http.StatusMethodNotAllowed, "method not allowed")
	}
}

func parseStackPath(path string) (string, string, error) {
	rest := strings.TrimPrefix(path, "/fops-router/v1/stacks/")
	parts := strings.Split(strings.Trim(rest, "/"), "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", fmt.Errorf("expected /fops-router/v1/stacks/{project}/{instance}")
	}

	project, err := url.PathUnescape(parts[0])
	if err != nil {
		return "", "", fmt.Errorf("invalid project path segment")
	}
	instance, err := url.PathUnescape(parts[1])
	if err != nil {
		return "", "", fmt.Errorf("invalid instance path segment")
	}

	return project, instance, nil
}

func decodeJSONBody(w http.ResponseWriter, r *http.Request, value any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return err
	}
	if err := decoder.Decode(&struct{}{}); err != io.EOF {
		return fmt.Errorf("request body must contain a single JSON document")
	}

	return nil
}

func writeJSON(w http.ResponseWriter, value any) error {
	w.Header().Set("Content-Type", "application/json")
	return json.NewEncoder(w).Encode(value)
}

func apiError(status int, message string) caddy.APIError {
	return caddy.APIError{
		HTTPStatus: status,
		Err:        fmt.Errorf("%s", message),
	}
}

func mutationAPIError(r *http.Request, err error) caddy.APIError {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return apiError(http.StatusBadRequest, err.Error())
	}

	return internalAPIError(r, err)
}

func internalAPIError(r *http.Request, err error) caddy.APIError {
	caddy.Log().Named("admin.api.fops_router").Error(
		"request failed",
		zap.String("method", r.Method),
		zap.String("path", r.URL.Path),
		zap.Error(err),
	)

	return apiError(http.StatusInternalServerError, "internal server error")
}

var _ caddy.AdminRouter = (*AdminHandler)(nil)
