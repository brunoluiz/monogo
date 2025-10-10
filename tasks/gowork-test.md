You have implemented what is in ./gowork.md. Now, implement tests to verify that it works as expected.

You should add a few tests using go.work, for example:

1. No changes detected
2. Change in module A causes an entry point module B using it to be marked as changed, while other entry points remain unchanged.
3. Change in go.mod version in the module triggers change detection in all entry points using that module.
4. Change in go.mod version in the module triggers change detection in all dependent modules.
5. Should detect a new file got created within the module and trigger entry points using it within that module as changed.
