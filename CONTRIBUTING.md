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

We have a script to update the vCluster dependency and vCluster CLI versions in E2E tests. Here's how to use it:

1. Clone the repository
2. `export VCLUSTER_VERSION="v0.26.0"`
3. Run `go run ./hack/bump-vcluster-dep.go "v0.26.0"` (replace "v0.26.0" with targeted vCluster version)
4. Add and commit changes to the new branch
5. Open a Pull Request

### After the PR is created:
- Verify that all E2E tests pass on the PR
- Some cases may require additional manual changes (if tests fail, check the logs)
- Once all tests pass, the PR is ready to be reviewed and merged
- merge the PR

### Update examples
Run following commands in each: `examples/bootstrap-with-deployment`, `examples/crd-sync`, `examples/hooks`:
1. `go get github.com/loft-sh/vcluster@${VCLUSTER_VERSION}`
2. Get the last commit SHA on main branch (one from the PR you just merged)
3. Run `go get github.com/loft-sh/vcluster-sdk@{COMMIT_SHA}`
4. Run `go mod tidy`
5. Ensure that example compiles by building a docker image: `docker build . -t my-repo/my-plugin:0.0.1`
6. If it fails to compile, you need to adjust the `*.go` files in the examples to the in the syncer interfaces in the vCluster targeted version
7. Once all of them compile, commit changes and open a PR
8. Merge PR

### Create vcluster-sdk release
1. In Github, click on "Create release", pick a tag and release branch, autogenerate release notes.
2. Release pipeline will build vcluster-sdk image and examples image

This process helps keep our vCluster dependencies up-to-date while ensuring compatibility through the e2e test suite.