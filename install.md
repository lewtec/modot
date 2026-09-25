# Install

Curl the setup script; it pulls the right GitHub Release binary for your platform:

```bash
curl -fsSL https://raw.githubusercontent.com/lewtec/modot/main/setup | bash
```

## Env vars the script reads

| Var | Default | Meaning |
|-----|---------|---------|
| `REPO` | `lewtec/modot` | `owner/repo` for releases |
| `APPNAME` | `modot` | binary name inside the archive |
| `VERSION` | `latest` | tag, or `latest` |
| `OS` | auto | `linux`, `darwin`, `windows` |
| `ARCH` | auto | `amd64`, `arm64`, `386` |
| `DOWNLOAD_DIR` | temp dir | where archives land |
| `GITHUB_TOKEN` | unset | raises API rate limits; uses `gh` auth if present |

Pin version/arch:

```bash
curl -fsSL https://raw.githubusercontent.com/lewtec/modot/main/setup | VERSION=v0.1.0 ARCH=arm64 bash
```
