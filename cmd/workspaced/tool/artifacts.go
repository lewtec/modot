package tool

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"text/tabwriter"

	"github.com/lewtec/lewkit/x/cmd"
	kittool "github.com/lewtec/lewkit/x/tool"
)

type Artifacts struct {
	Hint    cmd.StringArg `short:"H" long:"hint" help:"binary name hint for scoring (overrides the default derived from the package name)"`
	spec    cmd.StringArg
	version *cmd.StringArg
}

func (Artifacts) Description() string {
	return "List artifacts for a tool and rank them by ScoreArtifact " +
		"weight for the current platform"
}

var errNotArtifactTool = errors.New("resolved tool does not implement ArtifactTool")

func (a *Artifacts) Run(ctx context.Context) error {
	specStr := a.spec.Value()
	spec, err := kittool.Parse(specStr)
	if err != nil {
		return err
	}

	version := spec.Version
	if a.version != nil {
		version = a.version.Value()
	}

	backend, err := kittool.Get(spec.Backend)
	if err != nil {
		return err
	}
	installed, err := backend.Tool(spec.Package)
	if err != nil {
		return err
	}

	artifactTool, ok := installed.(kittool.ArtifactTool)
	if !ok {
		return fmt.Errorf("%w: %q", errNotArtifactTool, specStr)
	}

	artifacts, err := artifactTool.ListArtifacts(ctx, version)
	if err != nil {
		return err
	}

	effectiveHint := a.Hint.Value()
	if effectiveHint == "" {
		effectiveHint = filepath.Base(spec.Package)
	}

	type entry struct {
		kittool.Artifact

		Score int
	}

	entries := make([]entry, len(artifacts))
	for i, art := range artifacts {
		entries[i] = entry{
			Artifact: art,
			Score:    kittool.ScoreArtifact(art, runtime.GOOS, runtime.GOARCH, effectiveHint),
		}
	}

	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Score != entries[j].Score {
			return entries[i].Score > entries[j].Score
		}
		li := len(entries[i].URL)
		lj := len(entries[j].URL)
		if li != lj {
			return li < lj
		}
		return i < j
	})

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintf(os.Stdout, "# platform=%s/%s  hint=%q  version=%s  (0=ineligible)\n",
		runtime.GOOS, runtime.GOARCH, effectiveHint, version)
	fmt.Fprintln(w, "SCORE\tOS\tARCH\tURL\tHASH\tSIZE")

	for _, e := range entries {
		sizeStr := "-"
		if e.Size > 0 {
			sizeStr = fmt.Sprintf("%d", e.Size)
		}
		hashStr := e.Hash
		if hashStr == "" {
			hashStr = "-"
		}
		osStr := e.OS
		if osStr == "" {
			osStr = "-"
		}
		archStr := e.Arch
		if archStr == "" {
			archStr = "-"
		}
		fmt.Fprintf(w, "%d\t%s\t%s\t%s\t%s\t%s\n",
			e.Score, osStr, archStr, e.URL, hashStr, sizeStr)
	}
	return w.Flush()
}
