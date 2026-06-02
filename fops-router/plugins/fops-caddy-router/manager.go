package fopscaddyrouter

import (
	"encoding/json"
	"errors"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ConfigReloader func() error

type Manager struct {
	registryPath  string
	generatedPath string
	reloader      ConfigReloader
}

var registryCommitMu sync.Mutex

func NewManager(registryPath string, generatedPath string, reloader ConfigReloader) *Manager {
	if registryPath == "" {
		registryPath = defaultRegistryPath
	}
	if generatedPath == "" {
		generatedPath = defaultGeneratedPath
	}

	return &Manager{
		registryPath:  registryPath,
		generatedPath: generatedPath,
		reloader:      reloader,
	}
}

func (m *Manager) Status() (Status, error) {
	registry, err := m.LoadRegistry()
	if err != nil {
		return Status{}, err
	}

	status := Status{OK: true, StackCount: len(registry.Stacks)}
	for _, stack := range registry.Stacks {
		status.RouteCount += len(stack.Routes)
	}

	return status, nil
}

func (m *Manager) LoadRegistry() (Registry, error) {
	data, err := os.ReadFile(m.registryPath)
	if errors.Is(err, os.ErrNotExist) {
		return NewRegistry(), nil
	}
	if err != nil {
		return Registry{}, err
	}
	if len(data) == 0 {
		return NewRegistry(), nil
	}

	var registry Registry
	if err := json.Unmarshal(data, &registry); err != nil {
		return Registry{}, err
	}
	if registry.Stacks == nil {
		registry.Stacks = map[string]Stack{}
	}

	return registry, nil
}

func (m *Manager) UpsertStack(project string, instance string, stack Stack) (Registry, error) {
	registryCommitMu.Lock()
	defer registryCommitMu.Unlock()

	if err := validateStackIdentity(project, instance); err != nil {
		return Registry{}, err
	}
	if project != stack.Project || instance != stack.Instance {
		return Registry{}, validationErrorf("path project/instance does not match payload")
	}

	registry, err := m.LoadRegistry()
	if err != nil {
		return Registry{}, err
	}
	previous := cloneRegistry(registry)

	stack.UpdatedAt = time.Now().UTC()
	registry.Stacks[stackKey(project, instance)] = stack

	if err := m.commit(previous, registry); err != nil {
		return Registry{}, err
	}

	return registry, nil
}

func (m *Manager) DeleteStack(project string, instance string) (Registry, error) {
	registryCommitMu.Lock()
	defer registryCommitMu.Unlock()

	if err := validateStackIdentity(project, instance); err != nil {
		return Registry{}, err
	}

	registry, err := m.LoadRegistry()
	if err != nil {
		return Registry{}, err
	}
	previous := cloneRegistry(registry)
	delete(registry.Stacks, stackKey(project, instance))

	if err := m.commit(previous, registry); err != nil {
		return Registry{}, err
	}

	return registry, nil
}

func (m *Manager) commit(previous Registry, next Registry) error {
	if err := validateRegistry(next); err != nil {
		return err
	}

	previousRegistryData, _ := json.MarshalIndent(previous, "", "  ")
	previousGeneratedData, previousGeneratedErr := os.ReadFile(m.generatedPath)

	generated, err := GenerateCaddyfile(next)
	if err != nil {
		return err
	}
	registryData, err := json.MarshalIndent(next, "", "  ")
	if err != nil {
		return err
	}
	registryData = append(registryData, '\n')

	if err := atomicWrite(m.registryPath, registryData, 0644); err != nil {
		return err
	}
	if err := atomicWrite(m.generatedPath, generated, 0644); err != nil {
		_ = atomicWrite(m.registryPath, append(previousRegistryData, '\n'), 0644)
		return err
	}

	if m.reloader != nil {
		if err := m.reloader(); err != nil {
			_ = atomicWrite(m.registryPath, append(previousRegistryData, '\n'), 0644)
			if previousGeneratedErr == nil {
				_ = atomicWrite(m.generatedPath, previousGeneratedData, 0644)
			} else {
				_ = os.Remove(m.generatedPath)
			}
			return err
		}
	}

	return nil
}

func atomicWrite(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return err
	}

	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer func() {
		_ = os.Remove(tmpName)
	}()

	if n, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	} else if n != len(data) {
		_ = tmp.Close()
		return io.ErrShortWrite
	}
	if err := tmp.Chmod(mode); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}

	if err := os.Rename(tmpName, path); err != nil {
		return err
	}

	return syncParentDir(path)
}

func syncParentDir(path string) error {
	dir, err := os.Open(filepath.Dir(path))
	if err != nil {
		return err
	}
	defer func() {
		_ = dir.Close()
	}()

	return dir.Sync()
}

func cloneRegistry(registry Registry) Registry {
	cloned := NewRegistry()
	for key, stack := range registry.Stacks {
		routes := make([]Route, len(stack.Routes))
		copy(routes, stack.Routes)
		stack.Routes = routes
		cloned.Stacks[key] = stack
	}

	return cloned
}
