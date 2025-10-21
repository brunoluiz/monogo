<h1 align="center">monogo</h1>
<p align="center">🏗️ Golang (opinionated) mono-repository tooling</p>

## ✨ Features

1. No CLI or external dependency, everything within the `monogo` binary
2. Detect changes in multiple entrypoints (binaries/cmds) in a mono-repository
3. Detect changes based only on Go and embedded files your entrypoint depend on
4. Customised behaviour using `github.com/brunoluiz/monogo` package instead of the CLI

## ✋ Non-features

These are non-supported features at the moment, but it might change in the future.

1. It doesn't detect changes for tests
2. It doesn't detect changes in static files imported by Go code
3. It doesn't support mono-repositories with multiple go.mod file (it assumes the root one)

## 🕹️ Usage

In a mono-repository, you must run the following:

```sh
# Normal usage: you must pass the entry points for the binaries to be detected and the ref branch to compare
monogo detect --entrypoints './cmd/hello,./cmd/foo' --ref-branch refs/heads/my-branch

# Usage against another repository/folder
monogo detect --path ./some/folder/repo --entrypoints './cmd/hello,./cmd/foo' --ref-branch refs/heads/my-branch

# Show all entrypoints including unchanged ones
monogo detect --entrypoints './cmd/hello,./cmd/foo' --ref-branch refs/heads/my-branch --show-unchanged
```

The results will be in JSON format and can be used to trigger jobs to the changed
entrypoints. In the case below, only `./cmd/hello` needs to be re-built.

```json
{
  "changed": true,
  "git": {
    "hash": "18c61ae928daff98272ed3413a05738803718fb4",
    "ref": "refs/heads/my-branch",
    "files": {
      "created": { "all": ["created.go", "readme.md"], "go": ["created.go"] },
      "updated": { "all": ["updated.go"], "go": ["updated.go"] },
      "deleted": { "all": [], "go": [] },
      "impacted": { "all": ["updated.go", "created.go", "readme.md"], "go": ["created.go", "updated.go"] },
    }
  },
  "stats": {
    "started_at": "2025-09-03T18:37:58.661095+01:00",
    "ended_at": "2025-09-03T18:37:59.325769+01:00",
    "duration": 664
  },
  "entrypoints": [
    {
      "path": "./cmd/hello",
      "changed": true,
      "reasons": [
        "files changed",
        "files created/deleted",
        "dependencies changed",
        "go version changed",
        "no git changes"
      ]
    },
    {
      "path": "./cmd/foo",
      "changed": false,
      "reasons": []
    }
  ]
}
```

### Github Actions

Most likely you will want to run it in a `prepare` job so you can prepare a matrix later on. You must use it with `--output github` instead and set up similarly to this

#### Workflow

```yaml
  prepare:
    name: Prepare
    runs-on: ubuntu-latest
    timeout-minutes: 5
    outputs:
      monogo: ${{ steps.monogo.outputs.json }}
      entrypoints: ${{ steps.monogo.outputs.entrypoints }}
      impacted_files: ${{ steps.monogo.outputs.impacted_go_files }}
      changed: ${{ steps.monogo.outputs.changed }}
    steps:
      - uses: actions/checkout@v5
        with:
          fetch-depth: 0
          fetch-tags: true
      - uses: jdx/mise-action@v3
      - id: monogo
        run: make monogo | tee -a "$GITHUB_OUTPUT" "$GITHUB_STEP_SUMMARY"

  build:
    if: ${{ needs.prepare.outputs.changed == 'true' }}
    name: Build (${{ matrix.entrypoint.path }})
    runs-on: ubuntu-latest
    timeout-minutes: 10
    permissions:
      contents: write # required for trivy
      security-events: write # required for trivy
      packages: write # required for docker
      attestations: write # required for cosign
      id-token: write
    needs: [prepare, test]
    strategy:
      matrix:
        entrypoint: ${{ fromJSON(needs.prepare.outputs.entrypoints) }}
    steps: []
```

#### Makefile

```sh
git_base := $(if $(filter main,$(git_current_branch)),refs/remotes/origin/main~1,refs/remotes/origin/main)

.PHONY: monogo
monogo:
 @monogo detect --entrypoints $(shell find services -type d -name cmd -print0 \
 | xargs -0 -I {} find {} -maxdepth 1 -mindepth 1 -type d \
 | paste -sd ',' -) \
 --base-ref $(git_base) --compare-ref 'HEAD' --output github
```

## 📀 Install

### Linux and Windows

[Check the releases section](https://github.com/brunoluiz/monogo/releases) for more information details.

### MacOS

```
brew install brunoluiz/tap/monogo
```

### Other

```
go install github.com/brunoluiz/monogo/cmd/monogo@latest
```

## 📋 TODO

- [ ] Add support for detecting changes in tests
- [ ] Add support for mono-repositories with multiple go.mod files
