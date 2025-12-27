# kustomize-fzf

Interactively filter Kubernetes YAML manifests outputed by Kustomize with the power of `fzf`.

Intended usage is to combine with kustomize like:
```
kustomize build --enable-helm --load-restrictor=LoadRestrictionsNone | kustomize-fzf
```


Feed Kubernetes manifests (single or multi-document YAML) on stdin and select the ones you want to output. Display lines are concise and composable:

```
(apiVersion.)Kind (Namespace.)Name
```

By default ONLY `Kind Name` is shown. You can opt‑in to the extra prefixes:
- `--show-apiversion` adds `apiVersion.` in front of Kind when apiVersion is present
- `--show-namespace` adds `namespace.` in front of Name when namespace is present

Examples:
```
Deployment web
apps/v1.Deployment web                # with --show-apiversion
Deployment default.web                # with --show-namespace
apps/v1.Deployment default.web        # with both
```

## Features

- Reads multi-document Kubernetes YAML from stdin (`---` separators).
- Ignores empty and comment-only documents.
- Minimal, information-dense listing with opt-in prefixes.
- Colorized, pretty preview via `yq` (auto-detected) or raw fallback.
- Colorized output manifests (yq -C) unless color disabled; falls back to normalized plain YAML if color disabled or yq absent.
- Multi-selection support (`--multi`).
- Clean separation of parsing, UI wiring, and CLI.

## Requirements

- Go 1.25+
- [`fzf`](https://github.com/junegunn/fzf) in PATH (or set `kustomize-fzf_FZF_BIN`).
- Optional: [`yq`](https://github.com/mikefarah/yq) for formatting/preview.

## Install

```
go install github.com/TaliaMarine/kustomize-fzf@latest
```

### Migration Note

Prior to version v1.0.5 the module path and README used the misspelled organisation `TaliaMarine` (with an s). The GitHub remote is `TaliaMarine` (with a z). Update any imports or install commands from:

```
go install github.com/TaliaMarine/kustomize-fzf@v1.0.5
```

to:

```
go install github.com/TaliaMarine/kustomize-fzf@latest
```

If you prefer installing the command subdirectory explicitly (equivalent):

```
go install github.com/TaliaMarine/kustomize-fzf/cmd/kustomize-fzf@latest
```

Older tags (v1.0.0-v1.0.3) retain their original module path declaration and are not `go install`-able via the corrected spelling.

## Usage

```
# Basic (Kind Name)
cat manifests.yaml | kustomize-fzf

# Include apiVersion prefix before Kind
cat manifests.yaml | kustomize-fzf --show-apiversion

# Include namespace prefix before Name
cat manifests.yaml | kustomize-fzf --show-namespace

# Both prefixes
cat manifests.yaml | kustomize-fzf --show-apiversion --show-namespace

# Multi select
cat manifests.yaml | kustomize-fzf --multi --show-namespace

# Disable yq (raw preview & output)
cat manifests.yaml | kustomize-fzf --no-yq

# Show version
kustomize-fzf --version
```

## Preview & Output Formatting

If `yq` is found and neither the flag `--no-yq` nor env `kustomize-fzf_NO_YQ=1` is set:
- Preview uses `yq -C '.'` (color) unless `kustomize-fzf_NO_COLOR=1`.
- Output uses `yq -C '.'` (color) unless `kustomize-fzf_NO_COLOR=1`, then `yq -r '.'` to normalize without color.

Fallback: `cat` is used when `yq` is unavailable or disabled (no normalization). If you want normalized but uncolored output explicitly, set `kustomize-fzf_NO_COLOR=1` while keeping `yq` available.

## Flags

| Flag | Purpose |
|------|---------|
| `--multi` | Select multiple manifests |
| `--show-apiversion` | Prefix Kind with `apiVersion.` |
| `--show-namespace` | Prefix Name with `namespace.` |
| `--no-yq` | Disable use of `yq` for preview + output |
| `--version` | Print version and exit |

(Flag `--no-align` is a legacy no-op retained for compatibility; it no longer alters output.)

## Environment Variables

| Variable | Effect |
|----------|--------|
| `kustomize-fzf_FZF_BIN` | Override path to `fzf` |
| `kustomize-fzf_NO_YQ=1` | Force-disable `yq` usage |
| `kustomize-fzf_NO_COLOR=1` | Disable colored preview & final output (still normalizes with `yq -r` if available) |

## Exit Codes

- 0: Success
- Non-zero: Aborted selection, parse error, no objects, or missing `fzf`

## Development

```
make test
make build
```

Or with `just`:
```
just build
just install VERSION=1.0.0
```

## Release Automation

Managed by release-please. Conventional commits drive version bumps & changelog PRs. Tagging triggers multi-arch binaries.

### Building & Uploading Release Artifacts Manually

Because the GitHub `release` event is not used directly to avoid chained workflow limitations, artifact builds are performed manually:

1. Merge the release-please PR (this creates or updates the tag, e.g. `v1.2.3`).
2. Go to GitHub Actions > `release-build` workflow.
3. Click `Run workflow` and enter the tag (with or without leading `v`, both accepted).
4. The workflow checks out the tag, builds multi-arch artifacts, generates SHA256SUMS, and uploads them to the existing release.

Artifacts produced:
- `kustomize-fzf_<version>_linux_amd64.tar.gz`
- `kustomize-fzf_<version>_linux_arm64.tar.gz`
- `kustomize-fzf_<version>_darwin_amd64.tar.gz`
- `kustomize-fzf_<version>_darwin_arm64.tar.gz`
- `kustomize-fzf_<version>_windows_amd64.tar.gz`
- `SHA256SUMS`

If you need to rebuild (e.g., to fix a transient failure), re-run the workflow with the same tag; existing assets with identical names may need manual removal first.

## Conventional Commit Examples
```
feat(fzf): add optional apiVersion prefix
fix(parser): trim CRLF line endings
```

## Security Notes

Temporary per-object YAML files (0600) are created for preview and removed after the selection process completes.

## License

MIT (see LICENSE).

## Roadmap / Ideas

- Filter flags: `--kind`, `--namespace`, label selectors
- Additional preview fallbacks (e.g. `bat` highlighting)
- JSON metadata export / selection indices output
- Optional column-based legacy layout mode
- Auto-disable color when stdout is not a TTY (currently always color unless `kustomize-fzf_NO_COLOR`)

Contributions welcome!
