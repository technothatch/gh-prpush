# Contributing to gh-prpush

Thanks for helping improve a small `gh` extension.

## Ground rules

- Search [existing issues](https://github.com/technothatch/gh-prpush/issues) before opening a new one.
- One problem per issue, please.
- Be kind. This is maintained in spare time; assume good intent.

## Reporting a bug

Use the **Bug report** template. The two things that matter most:

1. The exact command you ran (`gh prpush`, `gh prpush --force-with-lease`, …).
2. The full error output, plus `gh --version` and your OS.

If the wrong remote or branch was pushed, say which PR number and which remotes
(`git remote -v`) you have — that's the logic under test.

## Suggesting a feature

Use the **Feature request** template. Describe the workflow you're stuck on, not
just the flag you want. The core behaviour should stay "ask `gh`, push, get out
of the way" — anything that needs persistent state is probably a different tool.

## Sending a pull request

1. Fork and branch from `main`.
2. Keep the change focused, and add or update tests — the remote-picking logic
   lives behind a fake command runner in `internal/app` and `internal/gitx`.
3. Run the checks locally:

   ```
   gofmt -l .
   go vet ./...
   go test ./...
   ```

4. Open a PR that references the issue it fixes and explains *why* the change is
   correct, not just what it does.

Small, focused PRs merge fastest.

## Releases

Maintainers cut releases; the workflow in `.github/workflows/ci.yml` builds the
platform binaries and attaches them to the release. Contributors don't need to
worry about versioning.
