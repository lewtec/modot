package deployer

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/lucasew/workspaced/internal/atomicfile"
	envdriver "github.com/lucasew/workspaced/pkg/driver/env"
	"github.com/lucasew/workspaced/pkg/filespine"
	"os"
	"sort"

	lewpath "github.com/lewtec/lewkit/x/path"
)

// StateStore is the interface for state persistence.
type StateStore interface {
	// Load loads the current state.
	Load() (*State, error)

	// Save persists the state.
	Save(state *State) error

	// Path returns the path or identifier of the store (for logging).
	Path() string
}

// FileStateStore implements StateStore using a JSON file.
// On disk, file keys are stored relative to Root (home for home apply,
// workspace/git root for codebase apply). In memory, Load returns absolute
// paths so planner/executor can use them directly.
type FileStateStore struct {
	path string
	root string
}

// NewFileStateStoreIn stores state at rel inside workspace.
// rel is a path inside the root, not a host path.
func NewFileStateStoreIn(workspace *lewpath.Root, rel lewpath.Path) (*FileStateStore, error) {
	parent := rel.Parent()
	if parent != lewpath.New(".") {
		if err := parent.MkdirAll(workspace, 0o755); err != nil {
			return nil, fmt.Errorf("create state directory: %w", err)
		}
	}
	opened, err := parent.OpenRoot(workspace)
	if err != nil {
		return nil, err
	}
	dir := opened.Name()
	if err := opened.Close(); err != nil {
		return nil, err
	}
	return &FileStateStore{
		path: filespine.HostPath(dir, lewpath.New(rel.Name())),
		root: workspace.Name(),
	}, nil
}

// NewFileStateStore creates a FileStateStore.
// root is the apply target base (e.g. $HOME or the workspace root); paths in
// the state file are stored relative to it. Empty root keeps absolute keys.
// A relative path is resolved inside root.
func NewFileStateStore(path, root string) (*FileStateStore, error) {
	expanded := envdriver.ExpandPath(path)
	name := lewpath.New(expanded)
	apply := envdriver.ExpandPath(root)
	if apply != "" && !name.IsAbs() {
		workspace, err := lewpath.Open(apply)
		if err != nil {
			return nil, err
		}
		defer workspace.Close()
		return NewFileStateStoreIn(workspace, name)
	}
	parent := name.Parent()
	if err := mkdirAbs(parent); err != nil {
		return nil, fmt.Errorf("create state directory: %w", err)
	}
	opened, err := filespine.OpenDir(parent)
	if err != nil {
		return nil, err
	}
	dir := opened.Name()
	if err := opened.Close(); err != nil {
		return nil, err
	}
	applyRoot := apply
	if apply != "" {
		if rootDir, err := lewpath.Open(apply); err == nil {
			applyRoot = rootDir.Name()
			rootDir.Close()
		}
	}
	return &FileStateStore{
		path: filespine.HostPath(dir, lewpath.New(name.Name())),
		root: applyRoot,
	}, nil
}

func mkdirAbs(dir lewpath.Path) error {
	if !dir.IsAbs() {
		return fmt.Errorf("state directory is not absolute")
	}
	slash, err := lewpath.Open("/")
	if err != nil {
		return err
	}
	defer slash.Close()
	rel, err := dir.Rel(lewpath.New("/"))
	if err != nil {
		return err
	}
	if rel == lewpath.New(".") {
		return nil
	}
	return rel.MkdirAll(slash, 0o755)
}

func (s *FileStateStore) Load() (*State, error) {
	state := &State{Files: make(map[string]ManagedInfo)}

	// If the file does not exist, return an empty state
	if _, err := os.Stat(s.path); errors.Is(err, os.ErrNotExist) {
		return state, nil
	}

	data, err := os.ReadFile(s.path)
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}

	var disk State
	if err := json.Unmarshal(data, &disk); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}

	if disk.Files == nil {
		return state, nil
	}

	// Expand on-disk keys (relative to root, ~/…, or legacy absolute) to abs.
	for key, info := range disk.Files {
		abs := AbsFromRoot(key, s.root)
		state.Files[abs] = info
	}

	return state, nil
}

func (s *FileStateStore) Save(state *State) error {
	disk := &State{Files: make(map[string]ManagedInfo)}
	if state != nil && state.Files != nil {
		// Deterministic key order for stable JSON (map range is random).
		keys := make([]string, 0, len(state.Files))
		for k := range state.Files {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, abs := range keys {
			rel := RelToRoot(abs, s.root)
			disk.Files[rel] = state.Files[abs]
		}
	}

	data, err := json.MarshalIndent(disk, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}

	if err := atomicfile.WriteBytes(s.path, data, 0o644); err != nil {
		return fmt.Errorf("write state file: %w", err)
	}
	return nil
}

func (s *FileStateStore) Path() string {
	return s.path
}

// MemoryStateStore implements StateStore in memory (useful for tests).
type MemoryStateStore struct {
	state *State
	id    string
}

// NewMemoryStateStore creates a MemoryStateStore.
func NewMemoryStateStore(id string) *MemoryStateStore {
	return &MemoryStateStore{
		state: &State{Files: make(map[string]ManagedInfo)},
		id:    id,
	}
}

func (s *MemoryStateStore) Load() (*State, error) {
	if s.state == nil {
		s.state = &State{Files: make(map[string]ManagedInfo)}
	}
	return s.state, nil
}

func (s *MemoryStateStore) Save(state *State) error {
	s.state = state
	return nil
}

func (s *MemoryStateStore) Path() string {
	return fmt.Sprintf("memory:%s", s.id)
}
