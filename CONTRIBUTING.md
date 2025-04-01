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

## Upgrade vCluster dependency

There is automated CI job to bump vCluster dependency & vCluster CLI version in E2E jobs.
In this repository, navigate to Actions -> Select "Update vCluster dep"  job and pass target vCluster version (e.g. `v0.25.0`) as an input.
CI job will generate a Pull Request in this repository. Make sure that e2e tests pass on this PR (there may be cases when additional, manual changes are needed) and once it's all green, you can merge this PR.
