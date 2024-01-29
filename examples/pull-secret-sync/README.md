## Image Pull Secret Sync Plugin

This example plugin syncs pull secrets from the host cluster into vcluster. The pull secrets are synced from the namespace where vcluster is installed into the default namespace in vcluster.

For more information how to develop plugins in vcluster, please refer to the [official vcluster docs](https://www.vcluster.com/docs/plugins/overview).

## Using the Plugin

To use the plugin, create a new vcluster with the `plugin.yaml`:

```
# Use public plugin.yaml
vcluster create my-vcluster -n my-vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/pull-secret-sync/plugin.yaml
```

This will create a new vcluster with the plugin installed. After that, wait for vcluster to start up and check:

```
# Create a image pull secret in the host namespace
kubectl create secret generic regcred \
    -n my-vcluster \
    --from-file=.dockerconfigjson=$HOME/.docker/config.json \
    --type=kubernetes.io/dockerconfigjson

# Check if it was synced to the vcluster
vcluster connect my-vcluster -n my-vcluster -- kubectl get secrets
```

## Building the Plugin

To just build the plugin image and push it to the registry, run:
```
# Build
docker build . -t my-repo/my-plugin:0.0.1

# Push
docker push my-repo/my-plugin:0.0.1
```

Then exchange the image in the `plugin.yaml`.

## Development

General vcluster plugin project structure:
```
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

Before starting to develop, make sure you have installed the following tools on your computer:
- [docker](https://docs.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) with a valid kube context configured
- [helm](https://helm.sh/docs/intro/install/), which is used to deploy vcluster and the plugin
- [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup) v0.6.0 or higher
- [DevSpace](https://devspace.sh/cli/docs/quickstart), which is used to spin up a development environment

After successfully setting up the tools, start the development environment with:
```
devspace dev -n vcluster
```

After a while a terminal should show up with additional instructions. Enter the following command to start the plugin:
```
go build -mod vendor -o plugin main.go && /vcluster/syncer start
```

You can now change a file locally in your IDE and then restart the command in the terminal to apply the changes to the plugin.

Delete the development environment with:
```
devspace purge -n vcluster
```
