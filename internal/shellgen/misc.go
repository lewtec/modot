package shellgen

import (
	"context"
	"fmt"

	envdriver "github.com/lewtec/modot/internal/driver/env"
)

// GenerateDaemon generates daemon startup code
func GenerateDaemon() (string, error) {
	return `# Start modot daemon if available
if command -v modot >/dev/null 2>&1; then
	(modot daemon --try &) &>/dev/null
fi
`, nil
}

// GenerateFlags generates shell init flags
func GenerateFlags(ctx context.Context) (string, error) {
	root, err := envdriver.GetDotfilesRoot(ctx)
	if err != nil {
		return "", fmt.Errorf("dotfiles root: %w", err)
	}
	return fmt.Sprintf(`# Flag to indicate modot shell init is being used
export MODOT_SHELL_INIT=1
export SD_ROOT=%q/bin
export DOTFILES=%q
export NIXCFG_ROOT_PATH=%q
`, root, root, root), nil
}
