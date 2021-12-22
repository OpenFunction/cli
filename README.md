# ![OpenFunctionCtl](docs/images/logo.png)

`cli` is the Command-Line Interface for [OpenFunction](https://github.com/OpenFunction/OpenFunction)

The `cli` repo is used to track issues for the `OpenFunction`.  this tool allows users to focus on the core functionality of the OpenFunction, while also presenting the relationship between the OpenFunction and its dependent components in a more abstract and friendly way to the user.

## Main commands
The main commands supported by the CLI are:
- init: provides management for openfunction’s framework.
- install: install OpenFunction and its dependencies.
- uninstall: uninstall OpenFunction and its dependencies.
- create: create a function from a file or stdin.
- apply: apply a function from a file or stdin.
- get: prints a table of the most important information about the specified function.
  - get builder: prints important information about the builder.
  - get serving:prints important information about the serving.
- delete: delete a specified the function.

## Getting started

Visit [ofn release](https://github.com/OpenFunction/cli/releases/) to select the `ofn` cli to deploy to your cluster.

### Use ofn to deploy OpenFunction

> We assume that you have placed the artifacts from the above step under the appropriate path in `PATH` and renamed it `ofn`. 

You can use the `ofn install --all` to complete a simple deployment. By default, this command will install the *v0.4.0* version of OpenFunction for you, while it will skip the installation process for components that already exist (you can use the `--upgrade` command to overwrite these components).

Please refer to the [ofn install docs](docs/install.md).

```shell
# ofn install --all --upgrade
Start installing OpenFunction and its dependencies.
Here are the components and corresponding versions to be installed:
+------------------+---------+
| COMPONENT        | VERSION |
+------------------+---------+
| Kourier          | 1.0.1   |
| Keda             | 2.4.0   |
| Tekton Pipelines | 0.30.0  |
| OpenFunction     | 0.4.0   |
| Dapr             | 1.5.1   |
| CertManager      | 1.1.0   |
| Shipwright       | 0.6.1   |
| Knative Serving  | 1.0.1   |
| DefaultDomain    | 1.0.1   |
+------------------+---------+
You have used the `--upgrade` parameter, which means that the installation process will overwrite the components that already exist.
Make sure you know what happens when you do this.
Enter 'y' to continue and 'n' to abort:
-> y
🔄  -> INGRESS <- Installing Ingress...
🔄  -> KNATIVE <- Installing Knative Serving...
🔄  -> DAPR <- Installing Dapr...
🔄  -> DAPR <- Downloading Dapr Cli binary...
🔄  -> KEDA <- Installing Keda...
🔄  -> CERTMANAGER <- Installing Cert Manager...
🔄  -> SHIPWRIGHT <- Installing Shipwright...
🔄  -> INGRESS <- Checking if Ingress is ready...
🔄  -> KEDA <- Checking if Keda is ready...
🔄  -> CERTMANAGER <- Checking if Cert Manager is ready...
🔄  -> SHIPWRIGHT <- Checking if Shipwright is ready...
🔄  -> KNATIVE <- Installing Kourier as Knative's gateway...
🔄  -> KNATIVE <- Configuring Knative Serving's DNS...
🔄  -> KNATIVE <- Checking if Knative Serving is ready...
✅  -> CERTMANAGER <- Done!
🔄  -> DAPR <- Initializing Dapr with Kubernetes mode...
✅  -> SHIPWRIGHT <- Done!
✅  -> KNATIVE <- Done!
✅  -> INGRESS <- Done!
✅  -> DAPR <- Done!
✅  -> KEDA <- Done!
🔄  -> OPENFUNCTION <- Installing OpenFunction...
🔄  -> OPENFUNCTION <- Checking if OpenFunction is ready...
✅  -> OPENFUNCTION <- Done!
🚀 Completed in 2m3.638035129s.
```

### Use ofn to uninstall OpenFunction

> We assume that you have placed the artifacts from the above step under the appropriate path in `PATH` and renamed it `ofn`. 

You can use `ofn uninstall --all` to easily uninstall OpenFunction and its dependencies (or just uninstall OpenFunction without arguments).

Please refer to the [ofn uninstall docs](docs/uninstall.md).

```shell
~# ofn uninstall --all
Start uninstalling OpenFunction and its dependencies.
The following components already exist:
+------------------+---------+
| COMPONENT        | VERSION |
+------------------+---------+
| Cert Manager     | v1.5.4  |
| Ingress Nginx    | 1.1.0   |
| Tekton Pipelines | v0.28.1 |
| Shipwright       | 0.6.0   |
| OpenFunction     | v0.4.0  |
| Dapr             | 1.4.3   |
| Keda             | 2.4.0   |
| Knative Serving  | 0.26.0  |
+------------------+---------+
You can see the list of components to be uninstalled and the list of components already exist in the cluster.
Make sure you know what happens when you do this.
Enter 'y' to continue and 'n' to abort:
-> y
🔄  -> OPENFUNCTION <- Uninstalling OpenFunction...
🔄  -> KNATIVE <- Uninstalling Knative Serving...
🔄  -> DAPR <- Uninstalling Dapr with Kubernetes mode...
🔄  -> KEDA <- Uninstalling Keda...
🔄  -> SHIPWRIGHT <- Uninstalling Tekton Pipeline & Shipwright...
🔄  -> INGRESS <- Uninstalling Ingress...
🔄  -> CERTMANAGER <- Uninstalling Cert Manager...
✅  -> OPENFUNCTION <- Done!
✅  -> DAPR <- Done!
🔄  -> KNATIVE <- Uninstalling Kourier...
✅  -> KEDA <- Done!
✅  -> CERTMANAGER <- Done!
✅  -> KNATIVE <- Done!
✅  -> INGRESS <- Done!
✅  -> SHIPWRIGHT <- Done!
🚀 Completed in 1m21.683329262s.
```

### Build OpenFunction CLI

You can also use `make build` to build the OpenFunction CLI —— `ofn`.
When the command is executed, you can find the artifact in the `. /dist` directory.
Move it to the appropriate path in the `PATH` so that you can use it in your environment.

```shell
~# make build
go fmt ./...
/opt/openfunction/fn-cli/bin/goimports -w cmd/ pkg/ testdata/
go vet ./...
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build  -ldflags "-s -w -X 'main.goversion=go version go1.16.7 linux/amd64'" \
-o ./dist/fn_linux_amd64 cmd/main.go;
```

### 
