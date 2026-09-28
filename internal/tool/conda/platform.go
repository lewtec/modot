package conda

import (
	"fmt"
	"runtime"
)

const subdirNoarch = "noarch"

func currentSubdir() (string, error) {
	return hostSubdir(runtime.GOOS, runtime.GOARCH)
}

func hostSubdir(goos, goarch string) (string, error) {
	switch goos {
	case "linux":
		switch goarch {
		case "amd64":
			return "linux-64", nil
		case "arm64":
			return "linux-aarch64", nil
		case "386":
			return "linux-32", nil
		case "ppc64le":
			return "linux-ppc64le", nil
		case "s390x":
			return "linux-s390x", nil
		case "riscv64":
			return "linux-riscv64", nil
		}
	case "darwin":
		switch goarch {
		case "amd64":
			return "osx-64", nil
		case "arm64":
			return "osx-arm64", nil
		}
	case "windows":
		switch goarch {
		case "amd64":
			return "win-64", nil
		case "arm64":
			return "win-arm64", nil
		case "386":
			return "win-32", nil
		}
	}
	return "", fmt.Errorf("%w: %s/%s", ErrNoBuild, goos, goarch)
}
