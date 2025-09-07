# Task

Consider the file detect.go:

I want to create a integration test against it, which:

1. Creates a folder with a git repository which will be used for testing
2. In this folder, create a little golang project. This can be created from a template which could live in a
`testdata` folder in the root.
3. This project must have a few internal packages it depends on, but as well a few external ones (eg: zap).
This project must have three different endpoints which have their own set of packages and some shared packages.
4. Do a few changes on the temporary repository, which must be committed to a separate branch
5. Run the detect package against it and evaluate the response from it

# Scenarios:

1. Project has a go version upgrade
2. Project has a go.mod dependency upgrade
3. Project internal package has a new file added
4. Project internal package has a file changed
5. Project internal package has a file deleted
6. Any other permutation you deem required

The test scenarios should test as well detecting changes only on one of the entrypoints and all entrypoints
when an internal dependency is changed (depends on what is being tested).

# Tips:

1. You should use go-git for most of git related tasks
2. You can use testing.T TempDir method for the temporary folder that needs creation
