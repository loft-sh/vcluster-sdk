## Hooks Plugin

This example plugin mutates objects vcluster is syncing to the host clsuter. 

For more information how to develop plugins in vcluster, please refer to the [official vcluster docs](https://www.vcluster.com/docs/plugins/overview).

## Using the Plugin

To use the plugin, create a new vcluster with the `plugin.yaml`:

```
# Use public plugin.yaml
vcluster create my-vcluster -n my-vcluster -f https://raw.githubusercontent.com/loft-sh/vcluster-sdk/main/examples/hooks/plugin.yaml
```

This will create a new vcluster with the plugin installed. After that, wait for vcluster to start up and check:

Create a file `deployment.yaml`:
```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: nginx-deployment
spec:
  selector:
    matchLabels:
      app: nginx
  replicas: 2 # tells deployment to run 2 pods matching the template
  template:
    metadata:
      labels:
        app: nginx
    spec:
      containers:
      - name: nginx
        image: nginx:1.14.2
        ports:
        - containerPort: 80
```

Then execute
```
# Create the deployment in the virtual cluster
vcluster connect my-vcluster -n my-vcluster -- kubectl apply -f deployment.yaml

# Check if pod label was set correctly and pod started
k get po -n my-vcluster -l created-by-plugin=pod-hook
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
├── Dockerfile.dev      # Development Dockerfile
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
