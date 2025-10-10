# Description

As a Golang engineer, I want to use this tool not only on simple mono-repos, but as well on mono-repos leveraging go workspaces, so that I can bump modules independently between services.

# Requirements

1. Make sure that the tool can detect if a go workspace is being used by looking for a `go.work` file in the root of the repository.
2. Make sure that the tool can parse the `go.work` file to identify all the modules included in the workspace.
3. Update the logic that identifies Go modules to handle multiple modules within a workspace.
4. It should consider that, when other modules within the `go.work` change, it might trigger a change in another module and it should be reported as changed
5. You must not try to create or edit things in parent directories
6. Your test bed is already set up in the `go-lab` folder and it must pass the acceptance criteria below

# Acceptance Criteria

- Is must return that `./services/hello-world/cmd/api` changed and `./services/hello-world/cmd/cli` did not change when running: `go run ./cmd/monogo detect --entrypoints './services/hello-world/cmd/api,./services/hello-world/cmd/cli' --path ./go-lab --base-ref 'refs/heads/test-gowork' --compare-ref 'refs/heads/test-gowork-change'`
- Disregard automated tests and linting (will be a separate task)
