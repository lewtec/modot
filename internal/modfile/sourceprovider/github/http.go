package github

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"

	httpclientdriver "github.com/lewtec/lewkit/x/driver/httpclient"
	githubprov "github.com/lewtec/lewkit/x/tool/github"
)

func (s Source) GetJSON(ctx context.Context, url string, out any) error {
	client, err := httpclientdriver.Client(ctx)
	if err != nil {
		return fmt.Errorf("get http client driver: %w", err)
	}

	req, err := githubprov.NewAPIRequest(ctx, http.MethodGet, url)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		if closer := resp.Body; closer != nil {
			if err := closer.Close(); err != nil {
				slog.Error("unexpected error", "op", "close", "error", err)
			}
		}
	}()
	if resp.StatusCode != http.StatusOK {
		hint := ""
		if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusNotFound || resp.StatusCode == http.StatusForbidden {
			if githubprov.Token(ctx) == "" {
				hint = " (private repos require GITHUB_TOKEN or 'gh auth login')"
			}
		}
		return fmt.Errorf("unexpected status: %s%s", resp.Status, hint)
	}
	return json.NewDecoder(resp.Body).Decode(out)
}
