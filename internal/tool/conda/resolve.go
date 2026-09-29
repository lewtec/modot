package conda

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/lewtec/lewkit/x/taskgroup"
)

// resolveState is one closure solve. Dependency fetches and artifact
// downloads are Control tasks; DownloadFile schedules the Internet work,
// and that pool applies backpressure.
type resolveState struct {
	ix       *index
	subdir   string
	download func(context.Context, record) error
	mu       sync.Mutex
	chosen   map[string]*choice
}

// constraint is one version and build request for a package.
type constraint struct {
	version string
	build   string
}

// choice is a package someone has already started resolving.
// ready closes once rec or err is published, before dependencies run,
// so a cycle can see the selection instead of waiting forever.
// done closes when the first walk returns.
// Later requests add constraints and may switch to another build that
// satisfies every one of them, which is how split packages such as
// libxml2 and libxml2-16 stay paired.
type choice struct {
	mu          sync.Mutex
	ready       chan struct{}
	done        chan struct{}
	base        string
	rec         record
	constraints []constraint
	gen         int
	err         error
	walkErr     error
}

type frameKey struct{}

func pushFrame(ctx context.Context, key string) context.Context {
	var parent []string
	if stack, ok := ctx.Value(frameKey{}).([]string); ok {
		parent = stack
	}
	next := make([]string, len(parent)+1)
	copy(next, parent)
	next[len(parent)] = key
	return context.WithValue(ctx, frameKey{}, next)
}

func frameHas(ctx context.Context, key string) bool {
	stack, _ := ctx.Value(frameKey{}).([]string)
	for _, item := range stack {
		if item == key {
			return true
		}
	}
	return false
}

func (ix *index) closure(ctx context.Context, ch channel, name, versionSpec, buildSpec, subdir string, download func(context.Context, record) error) ([]record, error) {
	state := &resolveState{
		ix:       ix,
		subdir:   subdir,
		download: download,
		chosen:   map[string]*choice{},
	}
	rootKey := ch.Base + "\x00" + strings.ToLower(strings.TrimSpace(name))
	err := taskgroup.WithSession(ctx, func(ctx context.Context) error {
		return state.walk(ctx, ch, name, versionSpec, buildSpec, true)
	})
	if err != nil {
		return nil, err
	}
	return state.finalRecords(rootKey)
}

func (state *resolveState) walk(ctx context.Context, ch channel, name, versionSpec, buildSpec string, root bool) (err error) {
	name = strings.ToLower(strings.TrimSpace(name))
	if name == "" || strings.HasPrefix(name, "__") {
		return nil
	}
	key := ch.Base + "\x00" + name
	if existing, ok := state.lookup(key); ok {
		return state.reconcile(ctx, ch, existing, key, name, versionSpec, buildSpec)
	}
	slot := &choice{ready: make(chan struct{}), done: make(chan struct{}), base: ch.Base}
	state.mu.Lock()
	if existing, ok := state.chosen[key]; ok {
		state.mu.Unlock()
		return state.reconcile(ctx, ch, existing, key, name, versionSpec, buildSpec)
	}
	state.chosen[key] = slot
	state.mu.Unlock()
	defer func() {
		slot.walkErr = err
		close(slot.done)
	}()

	signaled := false
	signal := func() {
		if signaled {
			return
		}
		signaled = true
		close(slot.ready)
	}
	defer signal()

	rec, err := state.selectRecord(ctx, ch, name, versionSpec, buildSpec, root)
	if err != nil {
		slot.mu.Lock()
		slot.err = err
		slot.mu.Unlock()
		signal()
		return err
	}
	slot.mu.Lock()
	slot.rec = rec
	slot.constraints = []constraint{{version: versionSpec, build: buildSpec}}
	gen := slot.gen
	slot.mu.Unlock()
	signal()
	return state.activate(ctx, slot, ch, key, rec, gen)
}

func (state *resolveState) lookup(key string) (*choice, bool) {
	state.mu.Lock()
	defer state.mu.Unlock()
	slot, ok := state.chosen[key]
	return slot, ok
}

func (state *resolveState) reconcile(ctx context.Context, ch channel, slot *choice, key, name, versionSpec, buildSpec string) error {
	select {
	case <-slot.ready:
	case <-ctx.Done():
		return context.Cause(ctx)
	}
	slot.mu.Lock()
	if slot.err != nil && slot.rec.Name == "" {
		err := slot.err
		slot.mu.Unlock()
		return err
	}
	slot.mu.Unlock()

	recs, err := state.ix.records(ctx, ch, name, state.subdir)
	if err != nil {
		return wrapMiss(name, versionSpec, buildSpec, err, false)
	}
	slot.mu.Lock()
	slot.constraints = append(slot.constraints, constraint{version: versionSpec, build: buildSpec})
	next, err := pickSatisfying(recs, slot.constraints, state.subdir)
	if err != nil {
		current := slot.rec
		slot.mu.Unlock()
		return fmt.Errorf("%w: %s %s conflicts with %s=%s", ErrUnsatisfied, name, strings.TrimSpace(versionSpec+" "+buildSpec), current.Version, current.Build)
	}
	next.URL = ch.artifactURL(next.Subdir, next.Filename)
	if sameArtifact(next, slot.rec) {
		slot.mu.Unlock()
		// An ancestor is already on this stack. Waiting for it would cycle.
		if frameHas(ctx, key) {
			return nil
		}
		select {
		case <-slot.done:
			return slot.walkErr
		case <-ctx.Done():
			return context.Cause(ctx)
		}
	}
	slot.gen++
	gen := slot.gen
	slot.rec = next
	slot.mu.Unlock()
	return state.activate(ctx, slot, ch, key, next, gen)
}

func (state *resolveState) activate(ctx context.Context, slot *choice, ch channel, key string, rec record, gen int) error {
	dlErr := state.startDownload(ctx, rec)
	deps, err := directDeps(rec.Depends)
	if err != nil {
		return err
	}
	ctx = pushFrame(ctx, key)
	depErr := state.fetchDeps(ctx, ch, rec.Name, deps)
	got, waitErr := state.waitDownload(ctx, dlErr)
	slot.mu.Lock()
	abandoned := slot.gen != gen
	slot.mu.Unlock()
	if abandoned {
		return depErr
	}
	err = preferError(depErr, got)
	if err == nil {
		err = waitErr
	}
	return err
}

func (state *resolveState) selectRecord(ctx context.Context, ch channel, name, versionSpec, buildSpec string, root bool) (record, error) {
	recs, err := state.ix.records(ctx, ch, name, state.subdir)
	if err != nil {
		return record{}, wrapMiss(name, versionSpec, buildSpec, err, root)
	}
	rec, err := pickRecord(recs, versionSpec, buildSpec, state.subdir)
	if err != nil {
		return record{}, wrapMiss(name, versionSpec, buildSpec, err, root)
	}
	rec.URL = ch.artifactURL(rec.Subdir, rec.Filename)
	return rec, nil
}

func wrapMiss(name, versionSpec, buildSpec string, err error, root bool) error {
	if root {
		return fmt.Errorf("%w: %s", err, name)
	}
	return fmt.Errorf("%w: %s %s: %w", ErrUnsatisfied, name, strings.TrimSpace(versionSpec+" "+buildSpec), err)
}

func (state *resolveState) startDownload(ctx context.Context, rec record) <-chan error {
	if state.download == nil {
		return nil
	}
	dlErr := make(chan error, 1)
	taskgroup.Go(ctx, "conda:download:"+rec.Name+"-"+rec.Version, taskgroup.Control, func(ctx context.Context, status *taskgroup.Status) error {
		status.Update(rec.Filename)
		err := state.download(ctx, rec)
		dlErr <- err
		return err
	})
	return dlErr
}

func (state *resolveState) fetchDeps(ctx context.Context, ch channel, name string, deps []string) error {
	if len(deps) == 0 {
		return nil
	}
	return taskgroup.Each[string]{
		Name:     "conda:fetch:" + name,
		Items:    deps,
		PoolKind: taskgroup.Control,
		TaskName: func(_ int, dep string) string {
			spec, err := parseMatchSpec(dep)
			if err != nil || spec.Name == "" {
				return "conda:fetch"
			}
			return "conda:fetch:" + spec.Name
		},
		Fn: func(ctx context.Context, _ *taskgroup.Status, dep string) error {
			return state.walkDep(ctx, ch, dep)
		},
	}.Run(ctx)
}

func (state *resolveState) waitDownload(ctx context.Context, dlErr <-chan error) (error, error) {
	if dlErr == nil {
		return nil, nil
	}
	select {
	case err := <-dlErr:
		return err, nil
	case <-ctx.Done():
		select {
		case err := <-dlErr:
			return err, nil
		default:
			return nil, context.Cause(ctx)
		}
	}
}

func (state *resolveState) walkDep(ctx context.Context, ch channel, dep string) error {
	spec, err := parseMatchSpec(dep)
	if err != nil {
		return err
	}
	if spec.Channel != "" {
		ch, err = resolveChannel(spec.Channel)
		if err != nil {
			return err
		}
	}
	return state.walk(ctx, ch, spec.Name, spec.Version, spec.Build, false)
}

func directDeps(depends []string) ([]string, error) {
	deps := make([]string, 0, len(depends))
	for _, dep := range depends {
		spec, err := parseMatchSpec(dep)
		if err != nil {
			return nil, err
		}
		if spec.Name == "" || strings.HasPrefix(spec.Name, "__") {
			continue
		}
		deps = append(deps, dep)
	}
	return deps, nil
}

// preferError keeps a real failure when the other error is only the
// cancellation that failure caused.
func preferError(primary, secondary error) error {
	if primary == nil {
		return secondary
	}
	if secondary == nil || !isCancel(primary) {
		return primary
	}
	return secondary
}

func isCancel(err error) bool {
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}

func (state *resolveState) finalRecords(rootKey string) ([]record, error) {
	visited := map[string]bool{}
	visiting := map[string]bool{}
	var out []record
	var walk func(string) error
	walk = func(key string) error {
		if visited[key] || visiting[key] {
			return nil
		}
		slot, ok := state.chosen[key]
		if !ok {
			return nil
		}
		visiting[key] = true
		slot.mu.Lock()
		rec := slot.rec
		base := slot.base
		slot.mu.Unlock()
		for _, dep := range rec.Depends {
			spec, err := parseMatchSpec(dep)
			if err != nil {
				return err
			}
			if spec.Name == "" || strings.HasPrefix(spec.Name, "__") {
				continue
			}
			depBase := base
			if spec.Channel != "" {
				depCh, err := resolveChannel(spec.Channel)
				if err != nil {
					return err
				}
				depBase = depCh.Base
			}
			if err := walk(depBase + "\x00" + spec.Name); err != nil {
				return err
			}
		}
		visiting[key] = false
		visited[key] = true
		slot.mu.Lock()
		out = append(out, slot.rec)
		slot.mu.Unlock()
		return nil
	}
	if err := walk(rootKey); err != nil {
		return nil, err
	}
	return out, nil
}

func sameArtifact(left, right record) bool {
	return left.Subdir == right.Subdir && left.Version == right.Version && left.Build == right.Build && left.Filename == right.Filename
}

func pickSatisfying(records []record, constraints []constraint, native string) (record, error) {
	matched := records
	for _, item := range constraints {
		next := make([]record, 0, len(matched))
		for _, rec := range matched {
			if rec.Subdir != native && rec.Subdir != subdirNoarch {
				continue
			}
			ok, err := matchVersion(item.version, rec.Version)
			if err != nil {
				return record{}, err
			}
			if ok && matchBuild(item.build, rec.Build) {
				next = append(next, rec)
			}
		}
		matched = next
		if len(matched) == 0 {
			if item.version == "" && item.build == "" {
				return record{}, ErrNoBuild
			}
			return record{}, fmt.Errorf("%w: %s", ErrNoBuild, strings.TrimSpace(item.version+" "+item.build))
		}
	}
	best := matched[0]
	for _, rec := range matched[1:] {
		if newerRecord(rec, best, native) {
			best = rec
		}
	}
	return best, nil
}

func pickRecord(records []record, versionSpec, buildSpec, native string) (record, error) {
	return pickSatisfying(records, []constraint{{version: versionSpec, build: buildSpec}}, native)
}

func newerRecord(a, b record, native string) bool {
	cmp := versionRank(a.Version, b.Version)
	if cmp != 0 {
		return cmp > 0
	}
	if a.BuildNumber != b.BuildNumber {
		return a.BuildNumber > b.BuildNumber
	}
	if nativeRank(a, native) != nativeRank(b, native) {
		return nativeRank(a, native) > nativeRank(b, native)
	}
	aConda := strings.HasSuffix(a.Filename, ".conda")
	bConda := strings.HasSuffix(b.Filename, ".conda")
	if aConda != bConda {
		return aConda
	}
	return a.Filename < b.Filename
}

func nativeRank(rec record, native string) int {
	if rec.Subdir == native {
		return 1
	}
	return 0
}

func versionRank(left, right string) int {
	cmp, err := compareVersions(left, right)
	if err != nil {
		return strings.Compare(left, right)
	}
	return cmp
}

func uniqueVersions(records []record, subdir string) []string {
	seen := map[string]struct{}{}
	versions := make([]string, 0, len(records))
	for _, rec := range records {
		if rec.Subdir != subdir && rec.Subdir != subdirNoarch {
			continue
		}
		if _, ok := seen[rec.Version]; ok {
			continue
		}
		seen[rec.Version] = struct{}{}
		versions = append(versions, rec.Version)
	}
	sort.Slice(versions, func(i, j int) bool {
		return versionRank(versions[i], versions[j]) > 0
	})
	return versions
}

func requestedSpec(version string) (string, string) {
	version = strings.TrimSpace(version)
	if version == "" || version == "latest" {
		return "", ""
	}
	if left, right, ok := strings.Cut(version, "="); ok && left != "" && right != "" && !strings.ContainsAny(left, "<>!~") {
		return "==" + left, right
	}
	if strings.ContainsRune("<>!=~", rune(version[0])) {
		return version, ""
	}
	return "==" + version, ""
}
