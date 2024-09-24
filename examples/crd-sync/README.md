# Custom Resource Definition Sync Plugin

This example plugin syncs a new resource `cars` from the virtual cluster to the
host cluster. It expects that this CRD was already installed in the host cluster.

For more information how to develop plugins in vcluster, please refer to the
[official vcluster docs](https://www.vcluster.com/docs/plugins/overview).

## Using the Plugin in vcluster

To use the plugin, create a new vcluster with the `plugin.yaml`:

```bash
# Apply cars crd in host cluster
kubectl apply -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/crd-sync/manifests/crds.yaml

# Create vcluster with plugin
vcluster create my-vcluster -n my-vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/crd-sync/plugin.yaml
```

This will create a new vcluster with the plugin installed. Then test the plugin with:

```bash
# Apply audi car to vcluster
vcluster connect my-vcluster -n my-vcluster -- kubectl apply -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/crd-sync/manifests/audi.yaml

# Check if car got correctly synced
kubectl get cars -n my-vcluster
```

## Building the Plugin

To just build the plugin image and push it to the registry, run:

```bash
# Build
docker build . -t my-repo/my-plugin:0.0.1

# Push
docker push my-repo/my-plugin:0.0.1
```

Then exchange the image in the `plugin.yaml`

## Development

General vcluster plugin project structure:

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
- [helm](https://helm.sh/docs/intro/install/), which is used to deploy vcluster
  and the plugin
- [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup) v0.20.0 or
  higher
- [DevSpace](https://devspace.sh/cli/docs/quickstart), which is used to spin up a
  development environment

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
