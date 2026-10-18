# gh-prpush

**`gh pr push` that the official `gh` CLI still doesn't have.**

[`gh pr push` has been requested since 2019](https://github.com/cli/cli/issues/2189)
(150+ upvotes, still open). Until it ships in `gh` proper, this extension fills the gap:

```
$ gh pr push
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

Requires the [GitHub CLI](https://cli.github.com/) (`gh`) on your PATH.

## Usage

```
cd your-repo
gh pr checkout 123     # or whatever workflow you use
# ... commit your changes ...
gh pr push
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
