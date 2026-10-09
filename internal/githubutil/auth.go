package githubutil

import (
	"context"
	"net/http"

	githubprov "github.com/lewtec/lewkit/x/tool/github"
)

// Token returns the github.com token resolved by lewkit.
func Token(ctx context.Context) string {
	return githubprov.Token(ctx)
}

// ApplyAuth sets Authorization when Token returns a token.
func ApplyAuth(ctx context.Context, req *http.Request) {
	if req == nil {
		return
	}
	if token := Token(ctx); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
}
