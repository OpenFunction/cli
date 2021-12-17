# ofn install

This command will help you to install OpenFunction and its dependencies.

## Parameters

```shell
--all                For installing all dependencies.
--async              For installing OpenFunction Async Runtime (Dapr & Keda).
--cert-manager       For installing Cert Manager.
--dapr               For installing Dapr.
--dry-run            Used to prompt for the components and their versions to be installed by the current command.
--ingress            For installing Ingress Nginx.
--keda               For installing Keda.
--knative            For installing Knative Serving (with Kourier as default gateway).
--region-cn          For users who have limited access to gcr.io or github.com.
--shipwright         For installing ShipWright.
--sync               For installing OpenFunction Sync Runtime (To be supported).
--upgrade            Upgrade components to target version while installing.
--verbose            Show verbose information.
--version string     Used to specify the version of OpenFunction to be installed. (default "v0.4.0")
--timeout duration   Set timeout time. Default is 5 minutes. (default 5m0s)
```

## Use Cases

### Install OpenFunction with a specific runtime

```shell
ofn install --async
```

or

```shell
ofn install --knative
```

### Install OpenFunction with limited access to gcr.io or github.com

```shell
ofn install --region-cn --all
```

> You'll need to add `--region-cn` to the uninstall cmd too if OpenFunction is installed with this flag.

### Overwrite installed components with default versions

```shell
ofn install --upgrade --all
```

### Install a specific version of OpenFunction

> default to v0.4.0 if no version specified

The available versions are:
- v0.3.1
- v0.4.0
- latest

```shell
ofn install --version v0.4.0
```

## The default compatibility matrix

OpenFunction relies on several components like Knative Serving, Dapr, Keda, Shipwright, and Tekton. Some of these components require a specified version of Kubernetes.

The OpenFunction CLI provides a default compatibility matrix based on which OpenFunction CLI will install a default selected version of each component for each version of kubernetes. 

The OpenFunction CLI keeps the installed component details in `$home/.ofn/inventory.yaml`.

| Components             | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20+ | CLI Option       | Description                                    |
| ---------------------- | --------------- | --------------- | --------------- | ---------------- | ---------------- | ---------------------------------------------- |
| Knative Serving        | 0.21.1          | 0.23.3          | 0.25.2          | 1.0.1            | `--knative`      | The synchronous function runtime               |
| Kourier                | 0.21.0          | 0.23.0          | 0.25.0          | 1.0.1            | `--knative`      | The default network layer for Knative          |
| Serving Default Domain | 0.21.0          | 0.23.0          | 0.25.0          | 1.0.1            | `--knative`      | The default DNS layout for Knative             |
| Dapr                   | 1.5.1           | 1.5.1           | 1.5.1           | 1.5.1            | `--async`        | The distributed application runtime of asynchronous function |
| Keda                   | 2.4.0           | 2.4.0           | 2.4.0           | 2.4.0            | `--async`        | The autoscaler of asynchronous function runtime|
| Shipwright             | 0.6.1           | 0.6.1           | 0.6.1           | 0.6.1            | `--shipwright`   | The function build framework                   |
| Tekton Pipelines       | 0.23.0          | 0.26.0          | 0.29.0          | 0.30.0           | `--shipwright`   | The function build pipeline                    |
| Cert Manager           | 1.5.4           | 1.5.4           | 1.5.4           | 1.5.4            | `--cert-manager` | OpenFunction webhook Certificate manager (For OpenFunction v0.4.0+ only). |
| Ingress Nginx          | na              | na              | 1.1.0           | 1.1.0            | `--ingress`      | Function ingress controller (For OpenFunction v0.4.0+ only). |

> The function ingress capability (i.e. OpenFunction Domain) can only be used in Kubernetes v1.19+.

## Customize components installation

To increase the flexibility of the installation, the OpenFunction CLI supports using environment variables to customize the versions of dependent components.

> Before using these environment variables, please make sure that there are no compatibility issues between the selected component and Kubernetes.
>
> And if you use environment variables during installation, please ensure that they are also present with the same value set during uninstallation.
>
> Please note that when installing a component, the customized component information will be obtained in the following order:
>
> ```
> yaml file environment variables > version environment variables
> ```

### Customize component version

The following are specs of component version environment variables. 

| Variable name            | Description                                                  | Example values |
| ------------------------ | ------------------------------------------------------------ | -------------- |
| DAPR_VERSION             | Version of Dapr                                              | 1.4.3, 1.5.1   |
| KEDA_VERSION             | Version of Keda                                              | 2.4.0, 2.5.0   |
| KNATIVE_SERVING_VERSION  | Version of Knative Serving                                   | 0.23.3, 1.0.1  |
| KOURIER_VERSION          | Version of Kourier                                           | 0.23.0, 0.26.0 |
| DEFAULT_DOMAIN_VERSION   | Version of Serving Default Domain                            | 0.23.0, 0.26.0 |
| SHIPWRIGHT_VERSION       | Version of Shipwright (Please do not use this environment variable for now) | 0.6.1          |
| TEKTON_PIPELINES_VERSION | Version of Tekton Pipelines                                  | 0.26.0, 0.29.0 |
| INGRESS_NGINX_VERSION    | Version of Ingress Nginx                                     | 1.1.0          |
| CERT_MANAGER_VERSION     | Version of Cert Manager                                      | 1.5.4          |

### Customize component yaml file

The following are specs of component yaml file environment variables. 

> You can use any value supported by kubectl's `--filename` option.

| Variable name             | Description                                                  |
| ------------------------- | ------------------------------------------------------------ |
| KEDA_YAML                 | Path of Keda yaml file                                       |
| KNATIVE_SERVING_CRD_YAML  | Path of Knative Serving crds yaml file                       |
| KNATIVE_SERVING_CORE_YAML | Path of Knative Serving core yaml file                       |
| KOURIER_YAML              | Path of Kourier yaml file                                    |
| DEFAULT_DOMAIN_YAML       | Path of Serving Default Domain yaml file                     |
| SHIPWRIGHT_YAML           | Path of Shipwright yaml file (Please do not use this environment variable for now) |
| TEKTON_PIPELINES_YAML     | Path of Tekton Pipelines yaml file                           |
| INGRESS_NGINX_YAML        | Path of Ingress Nginx yaml file                              |
| CERT_MANAGER_YAML         | Path of Cert Manager yaml file                               |
| OPENFUNCTION_YAML         | Path of OpenFunction yaml file                               |
