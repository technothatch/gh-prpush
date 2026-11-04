# gh-prpush

**`gh pr push` that the official `gh` CLI still doesn't have.**

[`gh pr push` has been requested since 2019](https://github.com/cli/cli/issues/2189)
(150+ upvotes, still open). Until it ships in `gh` proper, this extension fills the gap:

```
$ gh prpush
pushed HEAD to fork (fix-thing)
```

## What it does

1. Finds the PR for your current branch (`gh pr view`).
2. Picks the right remote:
   - **cross-repository PR** → the fork remote (matched by URL owner, then by remote name),
   - **same-repo PR** → `origin`.
3. Runs `git push <remote> HEAD:refs/heads/<head-branch>`.

That's it. No state, no config file — it asks `gh` and `git` and gets out of the way.

## Install

```
gh extension install technothatch/gh-prpush
```

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) on your PATH. Prebuilt
static binaries are attached to every release for Linux (amd64/arm64/386),
macOS (amd64/arm64) and Windows (amd64/arm64); `gh` picks the right one
automatically.

If you prefer to build from source:

```bash
git clone https://github.com/technothatch/gh-prpush.git
cd gh-prpush
go build -o gh-prpush .        # Windows: go build -o gh-prpush.exe .
gh extension install .
```

## Usage

> **Naming note:** this is a `gh` *extension*, so the command is
> **`gh prpush`** — not `gh pr push`. That's a `gh` limitation: extensions live
> in the `gh <name>` namespace, and the built-in `pr` command can't be extended
> with new subcommands. If/when the feature lands in `gh` itself
> ([cli/cli#2189](https://github.com/cli/cli/issues/2189)), it will be
> `gh pr push`.

```
cd your-repo
gh pr checkout 123     # or whatever workflow you use
# ... commit your changes ...
gh prpush
```

Flags:

| Flag | Effect |
|------|--------|
| `--force-with-lease` | push with `--force-with-lease` (safe-ish force push) |
| `-h`, `--help` | show help |

## Development

```
go test ./...
go build .
```

The remote-picking logic is fully covered by unit tests with a fake command runner.

## Related

- Upstream issue: [cli/cli#2189](https://github.com/cli/cli/issues/2189) — if/when `gh pr push` ships
  natively, uninstall this extension.

## License

[MIT](LICENSE)
