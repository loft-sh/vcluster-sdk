# Contributing to vcluster-sdk

## Build the Examples

Make sure you do a dual platform build for `linux/amd64` and `linux/arm64` via:

```
# Ensure docker builder with multi platform support
docker buildx create \                                                                                                                              
  --name container \
  --driver=docker-container

# Build & push image
docker buildx build --platform linux/amd64,linux/arm64 . -t ghcr.io/loft-sh/vcluster-example-hooks:v1 --builder container --push
```

## License

This project is licensed under the Apache 2.0 License.

## Copyright notice

It is important to state that you retain copyright for your contributions, but agree to license them for usage by the project and author(s) under the Apache 2.0 license. Git retains history of authorship, but we use a catch-all statement rather than individual names.

## Upgrading vCluster Dependency

We have an automated CI workflow to update the vCluster dependency and vCluster CLI versions in E2E tests. Here's how to use it:

1. Navigate to the "Actions" tab in the repository
2. Find and select the "Update vCluster dep" workflow
3. Click "Run workflow"
4. Enter the target vCluster version (e.g., `v0.25.0`) as the input parameter
5. Run the workflow

The CI job will automatically:
- Update the necessary dependencies
- Generate a Pull Request in this repository with the changes

### After the PR is created:
- Verify that all E2E tests pass on the PR
- Some cases may require additional manual changes (if tests fail, check the logs)
- Once all tests pass, the PR is ready to be reviewed and merged

This process helps keep our vCluster dependencies up-to-date while ensuring compatibility through the e2e test suite.