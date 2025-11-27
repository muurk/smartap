# Releasing

This guide covers how to create releases for the Smartap project.

## Overview

Releases are triggered by pushing a version tag to GitHub. When a tag matching the pattern `v*` is pushed, GitHub Actions automatically:

1. Builds binaries for all supported platforms
2. Generates a changelog from commit history
3. Creates a GitHub Release with all binaries attached
4. Includes SHA256 checksums for verification

## Supported Platforms

Each release produces binaries for:

| Platform | Architecture | Binary Suffix | Notes |
|----------|--------------|---------------|-------|
| Linux | x86_64 | `linux-amd64` | Standard servers, most VPS |
| Linux | ARM64 | `linux-arm64` | Raspberry Pi 4/5 (64-bit OS) |
| Linux | ARMv7 | `linux-armv7` | Raspberry Pi 3/Zero 2W (32-bit OS) |
| macOS | x86_64 | `darwin-amd64` | Intel Macs |
| macOS | ARM64 | `darwin-arm64` | Apple Silicon (M1/M2/M3) |
| Windows | x86_64 | `windows-amd64.exe` | Windows 10/11 |

Each release includes three binaries per platform:

- `smartap-server` - The main server application
- `smartap-cfg` - Configuration wizard utility
- `smartap-jtag` - JTAG/certificate flashing utility

## Version Format

Versions must follow semantic versioning with a `v` prefix:

```
v1.0.0        # Stable release
v1.2.3        # Patch release
v2.0.0-beta.1 # Pre-release (beta)
v2.0.0-rc1    # Pre-release (release candidate)
```

Pre-release versions (containing `-`) are automatically marked as pre-release on GitHub.

## Creating a Release

### Using Make (Recommended)

The easiest way to create a release:

```bash
make release VERSION=v1.0.0
```

This command will:

1. Validate the version format
2. Check for uncommitted changes (and abort if found)
3. Prompt for confirmation
4. Create an annotated git tag
5. Push the tag to GitHub

Example output:

```
Creating release v1.0.0...

Create and push tag v1.0.0? [y/N] y
Tag v1.0.0 created locally
Tag v1.0.0 pushed. GitHub Actions will create the release.
Check: https://github.com/muurk/smartap/actions
```

### Manual Method

If you prefer to create tags manually:

```bash
# Create annotated tag
git tag -a v1.0.0 -m "Release v1.0.0"

# Push tag to GitHub
git push origin v1.0.0
```

### Non-Interactive Method

For CI/CD or scripting, use the `tag` target which doesn't prompt:

```bash
make tag VERSION=v1.0.0
```

## Pre-Release Checklist

Before creating a release:

- [ ] All tests pass (`make test`)
- [ ] Code is formatted (`make fmt`)
- [ ] No linter warnings (`make lint`)
- [ ] All changes are committed
- [ ] You're on the `main` branch
- [ ] Documentation is up to date

## Testing Release Builds Locally

To verify binaries compile for all platforms without creating a release:

```bash
make release-dry-run
```

This builds all 18 binaries locally in `bin/release/`:

```
bin/release/
├── smartap-cfg-darwin-amd64
├── smartap-cfg-darwin-arm64
├── smartap-cfg-linux-amd64
├── smartap-cfg-linux-arm64
├── smartap-cfg-linux-armv7
├── smartap-cfg-windows-amd64.exe
├── smartap-jtag-darwin-amd64
├── smartap-jtag-darwin-arm64
├── smartap-jtag-linux-amd64
├── smartap-jtag-linux-arm64
├── smartap-jtag-linux-armv7
├── smartap-jtag-windows-amd64.exe
├── smartap-server-darwin-amd64
├── smartap-server-darwin-arm64
├── smartap-server-linux-amd64
├── smartap-server-linux-arm64
├── smartap-server-linux-armv7
└── smartap-server-windows-amd64.exe
```

## After Release

Once the tag is pushed:

1. **Monitor the build**: Check [GitHub Actions](https://github.com/muurk/smartap/actions) for build progress
2. **Verify the release**: Once complete, check the [Releases page](https://github.com/muurk/smartap/releases)
3. **Update documentation**: If needed, update the download page with the new version

The release workflow typically completes in 2-3 minutes.

## Troubleshooting

### "Working directory has uncommitted changes"

Commit or stash your changes before releasing:

```bash
git status              # See what's uncommitted
git add . && git commit # Commit changes
# or
git stash               # Temporarily stash changes
```

### "VERSION must be in format vX.Y.Z"

Ensure your version starts with `v` and follows semver:

```bash
# Wrong
make release VERSION=1.0.0
make release VERSION=version1

# Correct
make release VERSION=v1.0.0
make release VERSION=v1.0.0-beta.1
```

### Tag already exists

If you need to recreate a release (not recommended):

```bash
# Delete local tag
git tag -d v1.0.0

# Delete remote tag
git push origin :refs/tags/v1.0.0

# Recreate
make release VERSION=v1.0.0
```

!!! warning
    Deleting and recreating tags can cause confusion for users who already downloaded the release. Only do this for unreleased or broken releases.

### Build fails in GitHub Actions

1. Check the [Actions tab](https://github.com/muurk/smartap/actions) for error details
2. Common issues:
    - Go compilation errors (fix code, delete tag, re-release)
    - Missing dependencies (update `go.mod`)
    - Test failures (tests run before release build)

## Release Assets

Each GitHub Release includes:

| File | Description |
|------|-------------|
| `smartap-*-linux-amd64` | Linux x86_64 binaries |
| `smartap-*-linux-arm64` | Linux ARM64 binaries |
| `smartap-*-linux-armv7` | Linux ARMv7 binaries |
| `smartap-*-darwin-amd64` | macOS Intel binaries |
| `smartap-*-darwin-arm64` | macOS Apple Silicon binaries |
| `smartap-*-windows-amd64.exe` | Windows binaries |
| `checksums.txt` | SHA256 checksums for all files |

### Verifying Downloads

Users can verify downloaded binaries:

```bash
# Download checksums
curl -LO https://github.com/muurk/smartap/releases/download/v1.0.0/checksums.txt

# Verify a binary
sha256sum -c checksums.txt --ignore-missing
```
