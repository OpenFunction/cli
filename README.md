# ![OpenFunctionCtl](docs/images/logo.png)

`cli` is the command-line interface for [OpenFunction](https://github.com/OpenFunction/OpenFunction).

The `cli` repo is used to track issues for the `OpenFunction`. This tool allows you to focus on the core functionality of the OpenFunction, while also presents the relationship between the OpenFunction and its dependent components in a more abstract and friendly way.

## Main commands
The main commands supported by the CLI are:
- init: provides management for openfunction’s framework.
- install: installs OpenFunction and its dependencies.
- uninstall: uninstalls OpenFunction and its dependencies.
- create: creates a function from a file or stdin.
- apply: applies a function from a file or stdin.
- get: prints a table of the most important information about the specified function.
  - get builder: prints important information about the builder.
  - get serving: prints important information about the serving.
- delete: deletes the specified function.

## Getting started

Visit [ofn releases page](https://github.com/OpenFunction/cli/releases/) to download the `ofn` cli to deploy to your cluster.

### Use ofn to deploy OpenFunction

> Make sure you put the artifacts from the above step under the appropriate path in `PATH` and rename it `ofn`. 

Run `ofn install --all` to implement a simple deployment. By default, this command will install the *latest stable* version of OpenFunction and skips the installation of components that already exist. To overwrite the existing components, use the `--upgrade` command. 

For more information, refer to the [ofn install document](docs/install.md).

```shell
# ofn install --all
Start installing OpenFunction and its dependencies.
The following components will be installed:
+------------------+---------+
| COMPONENT        | VERSION |
+------------------+---------+
| Knative Serving  | 1.0.1   |
| Tekton Pipelines | 0.30.0  |
| OpenFunction     | 0.5.0   |
| Kourier          | 1.0.1   |
| DefaultDomain    | 1.0.1   |
| Keda             | 2.4.0   |
| CertManager      | 1.5.4   |
| Dapr             | 1.5.1   |
| Shipwright       | 0.6.1   |
| IngressNginx     | 1.1.0   |
+------------------+---------+
 ✓ Dapr - Completed!
 ✓ Keda - Completed!
 ✓ Knative Serving - Completed!
 ✓ Shipwright - Completed!
 ✓ Cert Manager - Completed!
 ✓ Ingress - Completed!
 ✓ OpenFunction - Completed!
🚀 Completed in 1m40.055438303s.

 ██████╗ ██████╗ ███████╗███╗   ██╗
██╔═══██╗██╔══██╗██╔════╝████╗  ██║
██║   ██║██████╔╝█████╗  ██╔██╗ ██║
██║   ██║██╔═══╝ ██╔══╝  ██║╚██╗██║
╚██████╔╝██║     ███████╗██║ ╚████║
 ╚═════╝ ╚═╝     ╚══════╝╚═╝  ╚═══╝

███████╗██╗   ██╗███╗   ██╗ ██████╗████████╗██╗ ██████╗ ███╗   ██╗
██╔════╝██║   ██║████╗  ██║██╔════╝╚══██╔══╝██║██╔═══██╗████╗  ██║
█████╗  ██║   ██║██╔██╗ ██║██║        ██║   ██║██║   ██║██╔██╗ ██║
██╔══╝  ██║   ██║██║╚██╗██║██║        ██║   ██║██║   ██║██║╚██╗██║
██║     ╚██████╔╝██║ ╚████║╚██████╗   ██║   ██║╚██████╔╝██║ ╚████║
╚═╝      ╚═════╝ ╚═╝  ╚═══╝ ╚═════╝   ╚═╝   ╚═╝ ╚═════╝ ╚═╝  ╚═══╝
```

### Use ofn to uninstall OpenFunction

> Make sure you put the artifacts from the above step under the appropriate path in `PATH` and rename it `ofn`. 

Run `ofn uninstall --all` to uninstall OpenFunction and its dependencies (or just uninstall OpenFunction without arguments).

For more information, refer to the [ofn uninstall document](docs/uninstall.md).

```shell
~# ofn uninstall --all -y
Start uninstalling OpenFunction and its dependencies.
The following components already exist:
+------------------+---------+
| COMPONENT        | VERSION |
+------------------+---------+
| OpenFunction     | 0.5.0   |
| Knative Serving  | 1.0.1   |
| Kourier          | 1.0.1   |
| DefaultDomain    | 1.0.1   |
| IngressNginx     | 1.1.0   |
| Keda             | 2.4.0   |
| Dapr             | 1.5.1   |
| Shipwright       | 0.6.1   |
| Tekton Pipelines | 0.30.0  |
| CertManager      | 1.5.4   |
+------------------+---------+
 ✓ Dapr - Completed!
 ✓ Keda - Completed!
 ✓ Knative Serving - Completed!
 ✓ Shipwright - Completed!
 ✓ Tekton Pipelines - Completed!
 ✓ Cert Manager - Completed!
 ✓ Ingress - Completed!
 ✓ OpenFunction - Completed!
🚀 Completed in 1m17.729501739s.
```
