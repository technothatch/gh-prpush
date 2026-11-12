# gh-prpush

**`gh pr push` that the official `gh` CLI still doesn't have.**

[`gh pr push` has been requested since 2019](https://github.com/cli/cli/issues/2189)
(150+ upvotes, still open). Until it ships in `gh` proper, this extension fills the gap:

```
$ gh prpush
pushed HEAD to fork (fix-thing)
```

> **The story behind it:** [Push to the pull request you are actually working on](https://technothatch.iyre.com/blog/gh-prpush-inside-the-project/)
> — the TechnoThatch journal piece on why `gh pr push` is still missing.

## What it does

1. Finds the PR for your current branch (`gh pr view`).
2. Picks the right remote:
   - **cross-repository PR** → the fork remote (matched by URL owner, then by remote name),
   - **same-repo PR** → the repository's own remote, **whatever it is called**.
     A remote named `upstream` works just as well as `origin`, and when both
     exist — your fork as `origin`, the base repo as `upstream` — the push goes
     to the base repo.
3. Runs `git push <remote> HEAD:refs/heads/<head-branch>`.

That's it. No state, no config file — it asks `gh` and `git` and gets out of the way.

### When the fork has no remote yet

After `gh pr checkout` on a machine where the fork was never cloned, there is no
remote for the fork, and plain `git push` cannot know where to send the branch.
`gh prpush` fails with the exact command to run.

`--add-remote` does it for you:

```
$ gh prpush --add-remote
added remote somebody (https://github.com/somebody/gh-prpush.git)
pushed HEAD to somebody (fix-thing)
```

The remote is named after the fork owner and its URL comes from `gh`, so the
scheme matches the remotes you already have (HTTPS stays HTTPS — `gh`'s
credential helper is HTTPS-only by default).

It also works in the state where `gh pr view` itself cannot find the pull
request: `gh` resolves the PR for the current branch through git's push-target
configuration, which it cannot do when the fork is not a registered remote
([cli/cli#2942](https://github.com/cli/cli/issues/2942)). When that happens,
`gh prpush` falls back to `gh pr list --head`, which finds the same PR.

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
| `--add-remote` | add the fork as a remote if it isn't registered yet, then push |
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
- Related upstream request: [cli/cli#2942](https://github.com/cli/cli/issues/2942) — `gh pr checkout`
  should add the fork as a remote. `--add-remote` does that part for the push.

## License

[MIT](LICENSE)
