package github

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/lewtec/modot/internal/modfile"
)

const (
	gitTagPageSize = 100
	gitTagMaxPages = 50
)

var ErrTagListTruncated = errors.New("github tag list exceeded page cap")

type gitTag struct {
	Name   string `json:"name"`
	Commit struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func (s *Source) resolveTagSpec(ctx context.Context) error {
	pattern, ok, err := modfile.TagSpec(s.Config.Ref)
	if err != nil || !ok {
		return err
	}
	if s.Repo() == "" {
		return ErrRepoRequired
	}
	name, sha, err := s.newestTag(ctx, pattern)
	if err != nil {
		return fmt.Errorf("resolve tag pattern: %w", err)
	}
	s.Config.Ref = name
	s.Config.Versioning = modfile.TagVersioning(pattern)
	s.Config.URL = codeloadTarballURL(s.Repo(), sha)
	return nil
}

func (s Source) newestTag(ctx context.Context, pattern string) (string, string, error) {
	tags, err := s.listTags(ctx)
	if err != nil {
		return "", "", err
	}
	names := make([]string, 0, len(tags))
	shaByName := make(map[string]string, len(tags))
	for _, tag := range tags {
		name := strings.TrimSpace(tag.Name)
		if name == "" {
			continue
		}
		names = append(names, name)
		sha := strings.TrimSpace(tag.Commit.SHA)
		if prev, ok := shaByName[name]; !ok || prev == "" {
			shaByName[name] = sha
		}
	}
	name, err := modfile.SelectNewestTag(names, pattern)
	if err != nil {
		return "", "", err
	}
	sha := shaByName[name]
	if sha == "" {
		return "", "", ErrMissingSHA
	}
	return name, sha, nil
}

func (s Source) listTags(ctx context.Context) ([]gitTag, error) {
	repo := s.Repo()
	if repo == "" {
		return nil, ErrRepoRequired
	}
	var all []gitTag
	for page := 1; page <= gitTagMaxPages; page++ {
		endpoint := fmt.Sprintf("https://api.github.com/repos/%s/tags?per_page=%d&page=%d", repo, gitTagPageSize, page)
		var batch []gitTag
		if err := s.GetJSON(ctx, endpoint, &batch); err != nil {
			return nil, fmt.Errorf("list tags: %w", err)
		}
		all = append(all, batch...)
		if len(batch) < gitTagPageSize {
			return all, nil
		}
	}
	return nil, fmt.Errorf("list tags: %w", ErrTagListTruncated)
}
