# Import Secret Plugin
[DEPRECATED] 
Please use `sync.fromHost.secrets` with `sync.fromHost.secrets.mappings.byName` (added in v0.23) instead: https://www.vcluster.com/docs/vcluster/configure/vcluster-yaml/sync/from-host/secrets


This example plugin syncs all secrets with an annotation from the host cluster
into vCluster. The secrets are synced from the namespace where vCluster is
installed into the specified namespace in vCluster.

For more information how to develop plugins in vCluster, please refer to the
[official vCluster docs](https://www.vcluster.com/docs/plugins/overview).

## Using the Plugin

To use the plugin, create a new vCluster with the `plugin.yaml`:

```bash
# Use public plugin.yaml
vcluster create vcluster -n vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/import-secrets/plugin.yaml
```

This will create a new vCluster with the plugin installed. After that, wait for
vCluster to start up and check:

```bash
# Create a image pull secret in the host namespace
kubectl create secret generic test-secret \
    -n vcluster \
    --from-literal=my-key=my-value
    
# Tell vcluster to sync the secret to namespace test and name my-test within vCluster
kubectl annotate secret test-secret -n vcluster vcluster.loft.sh/import=test/my-test

# Check if it was synced to the vCluster
vcluster connect vcluster -n vcluster -- kubectl get secrets -A
```

## Building the Plugin

To just build the plugin image and push it to the registry, run:

```bash
# Build
docker build . -t my-repo/my-plugin:0.0.1

# Push
docker push my-repo/my-plugin:0.0.1
```

Then exchange the image in the `plugin.yaml`.

## Development

General vCluster plugin project structure:

```text
.
├── go.mod              # Go module definition
├── go.sum
├── devspace.yaml       # Development environment definition
├── devspace_start.sh   # Development entrypoint script
├── Dockerfile          # Production Dockerfile 
├── main.go             # Go Entrypoint
├── plugin.yaml         # Plugin Helm Values
├── syncers/            # Plugin Syncers
└── manifests/          # Additional plugin resources
```

Before starting to develop, make sure you have installed the following tools on
your computer:

- [docker](https://docs.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) with a valid kube context configured
- [helm](https://helm.sh/docs/intro/install/), which is used to deploy vCluster
  and the plugin
- [vCluster CLI](https://www.vcluster.com/docs/getting-started/setup) v0.20.0 or
  higher
- [DevSpace](https://devspace.sh/cli/docs/quickstart), which is used to spin up
  a development environment

After successfully setting up the tools, start the development environment with:

```bash
devspace dev -n vcluster
```

After a while a terminal should show up with additional instructions. Enter the
following command to start the plugin:

```bash
go build -mod vendor -o plugin main.go && /vcluster/syncer start
```

You can now change a file locally in your IDE and then restart the command in the
terminal to apply the changes to the plugin.

Delete the development environment with:

```bash
devspace purge -n vcluster
```
