This example plugin syncs pull secrets from the host cluster into vcluster. The pull secrets are synced from the namespace where vcluster is installed into the default namespace in vcluster.

For more information how to develop plugins in vcluster, please refer to the [official vcluster docs](https://www.vcluster.com/docs/what-are-virtual-clusters).

## Development

Before starting to develop, make sure you have installed the following tools on your computer:
- [docker](https://docs.docker.com/)
- [kubectl](https://kubernetes.io/docs/tasks/tools/) with a valid kube context configured
- [helm](https://helm.sh/docs/intro/install/), which is used to deploy vcluster and the plugin
- [vcluster CLI](https://www.vcluster.com/docs/getting-started/setup) v0.6.0 or higher
- [DevSpace](https://devspace.sh/cli/docs/quickstart), which is used to spin up a development environment

If you want to develop within a remote Kubernetes cluster (as opposed to docker-desktop or minikube), make sure to exchange `PLUGIN_IMAGE` in the `devspace.yaml` with a valid registry path you can push to.

After successfully setting up the tools, start the development environment with:
```
devspace dev -n vcluster
```

After a while a terminal should show up with additional instructions. Enter the following command to start the plugin:
```
go run -mod vendor main.go
```

The output should look something like this:
```
I0124 11:20:14.702799    4185 logr.go:249] plugin: Try creating context...
I0124 11:20:14.730044    4185 logr.go:249] plugin: Waiting for vcluster to become leader...
I0124 11:20:14.731097    4185 logr.go:249] plugin: Starting syncers...
[...]
I0124 11:20:15.957331    4185 logr.go:249] plugin: Successfully started plugin.
```

You can now change a file locally in your IDE and then restart the command in the terminal to apply the changes to the plugin.

Delete the development environment with:
```
devspace purge -n vcluster
```

## Using the Plugin in vcluster

### Building the Plugin
To just build the plugin image and push it to the registry, run:
```
# Build
docker build . -t my-repo/my-plugin:0.0.1

# Push
docker push my-repo/my-plugin:0.0.1
```

### Using the Plugin

To use the plugin, create a new vcluster with the `plugin.yaml`:

```
# Use local plugin.yaml
vcluster create my-vcluster -n my-vcluster -f ./plugin.yaml

# Use public plugin.yaml
vcluster create my-vcluster -n my-vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/pull-secret-sync/plugin.yaml
```

This will create a new vcluster with the plugin installed.
