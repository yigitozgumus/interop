# Release Process

This project uses GoReleaser for automating the release process.

## Regular Releases

To create a new regular release:

1. Create a new tag following semantic versioning:
   ```bash
   git tag -a v0.1.0 -m "First release"
   ```

2. Push the tag to the repository:
   ```bash
   git push origin v0.1.0
   ```

3. The GitHub Actions workflow will automatically build and publish the release artifacts.

## Snapshot Releases

For snapshot (development) releases, add `-snapshot` to the tag name:

```bash
git tag -a v0.1.0-snapshot -m "Snapshot release v0.1.0"
git push origin v0.1.0-snapshot
```

Snapshot releases will generate binaries with different configurations:

- Different binary names (with "_snapshot_" in the name)
- Different build parameters
- Flag set: `-X main.isSnapshot=true`

## Testing Locally

You can test builds locally without creating a tag using:

```bash
# Test a regular release
goreleaser release --snapshot --clean --skip=publish

# Test a snapshot release
GORELEASER_CURRENT_TAG=v0.1.0-snapshot goreleaser release --snapshot --clean --skip=publish
```