# one

`one` is a local-first CLI for managing this monorepo.

## Current commands

- `one projects sync [project]`
  - Crawls the repo for `one.yaml` files and writes `.one/projects.json`.
  - Optional `project` argument can be a project name, alias, directory, or `one.yaml` path to sync one entry.
  - Imports project `name`, `alias`, `desc`, and relative path from repo root.
- `one projects use -p <dir-or-alias>`
  - Sets default project in `.one/projects.json`.
- `one git remotes sync [project]` (or `-p <directory>`)
  - Adds or updates git remotes defined in the selected project's `one.yaml`.
  - With `--create`, creates missing GitHub repos via `gh` before adding remotes.
- `one git remotes add <name> <url>`
  - Adds or updates a single remote.
  - With `--create`, creates missing GitHub repo via `gh` when URL points to GitHub.
- `one git remotes list`
  - Lists current local git remotes and URLs.
- `one git sync [project]` (or `-p <directory>`)
  - Pushes to `origin`.
  - If `git.sync.subtree: true` (or `git.push.subtree: true`), pushes subtree from the selected project folder to configured remotes.
  - If `-p/--project` is omitted, `one` uses default project from `.one/projects.json`.

## Configuration

`one.yaml` example:

```yaml
name: JavaScript Library
alias: js
desc: JavaScript libraries and tooling subtree.

git:
  remotes:
    js: git@github.com:mrneobeard/js
    nb-js: git@github.com:neobeard/js
  sync:
    subtree: true
    remotes: [js, nb-js]
```

If `sync.remotes` is omitted, all keys from `git.remotes` are used.

## Build

From `apps/one`:

```bash
mise run build
```

This writes the executable to `../../bin/one`.

## Tests and quality

```bash
mise run test
mise run lint
mise run vuln
```

GitHub live integration test (creates and deletes a repo):

```bash
ONE_GH_INTEGRATION=1 go test -tags gh_integration ./cmd -run TestGitRemotesAddCreateGitHubRepo
```
