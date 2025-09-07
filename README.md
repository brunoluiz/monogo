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
# Normal usage: you must pass the entry points for the binaries to be detected
monogo detect --entrypoints './cmd/hello,./cmd/foo'

# Usage against another repository/folder
monogo detect --path ./some/folder/repo --entrypoints './cmd/hello,./cmd/foo'
```

The results will be in JSON format and can be used to trigger jobs to the changed
entrypoints. In the case below, only `./cmd/hello` needs to be re-built.

```json
{
  "git": {
    "hash": "18c61ae928daff98272ed3413a05738803718fb4",
    "ref": "refs/heads/my-branch"
  },
  "stats": {
    "started_at": "2025-09-03T18:37:58.661095+01:00",
    "ended_at": "2025-09-03T18:37:59.325769+01:00",
    "duration": 664
  },
  "entrypoints": {
    "./cmd/hello": {
      "path": "./cmd/hello",
      "changed": true,
      "reasons": [
        "files changed",
        "files created/deleted",
        "dependencies changed"
      ]
    },
    "./cmd/foo": {
      "path": "./cmd/foo",
      "changed": false,
      "reasons": []
    }
  }
}
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

- Standardised local/CI via Nix a
- Optimise the time spent on the `golang.org/x/tools/go/packages` module
- What is the minimum Go version it should use?
