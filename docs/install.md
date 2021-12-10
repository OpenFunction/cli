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
--region-cn          For users in China to speed up the download process of dependent components.
--shipwright         For installing ShipWright.
--sync               For installing OpenFunction Sync Runtime (To be supported).
--upgrade            Upgrade components to target version while installing.
--verbose            Show verbose information.
--version string     Used to specify the version of OpenFunction to be installed. The permitted versions are: v0.3.1, v0.4.0, latest. (default "v0.4.0")
--timeout duration   Set timeout time. Default is 5 minutes. (default 5m0s)
```

## Use Cases

#### Installing OpenFunction with a specify runtime

```shell
ofn install --async
```

or

```shell
ofn install --knative
```

#### Support for users in China to speed up the installation process

> This requires that you should also add the `--region-cn` parameter when executing the uninstall operation

```shell
ofn install --region-cn --all
```

#### Overwrite existing components in the cluster with the --upgrade parameter

```shell
ofn install --upgrade --all
```

#### Supports installation of multiple versions of OpenFunction

```shell
ofn install --version latest
```

## Inventory

Because different versions of OpenFunction depend on different lists of components, and some components have specific requirements for the version of the Kubernetes server at the same time.

The OpenFunction CLI provides a default installation inventory for each scenario. By default, you will complete the installation process based on the default installation inventory.

The OpenFunction CLI records the inventory of installed components in YAML format to the `$home/.ofn/inventory.yaml`.

### Default inventory under different versions of kubernetes

> The function ingress capability of OpenFunction (i.e. OpenFunction Domain) can only be used in Kubernetes v1.19+.

| Components             | Kubernetes 1.17 | Kubernetes 1.18 | Kubernetes 1.19 | Kubernetes 1.20+ | Description                                                  |
| ---------------------- | --------------- | --------------- | --------------- | ---------------- | ------------------------------------------------------------ |
| Knative Serving        | 0.21.1          | 0.23.3          | 0.25.2          | 1.0.1            | Controlled by the `--knative` option, provides synchronous function serving runtime capability |
| Kourier                | 0.21.0          | 0.23.0          | 0.25.0          | 1.0.1            | Controlled by the `--knative` option, as the default network layer for the Knative service |
| Serving Default Domain | 0.21.0          | 0.23.0          | 0.25.0          | 1.0.1            | Controlled by the `--knative` option, as the default DNS layout for the Knative service |
| Dapr                   | 1.5.1           | 1.5.1           | 1.5.1           | 1.5.1            | Controlled by the `--async` option, collaboration with KEDA to provide asynchronous function serving runtime capability |
| Keda                   | 2.4.0           | 2.4.0           | 2.4.0           | 2.4.0            | Controlled by the `--async` option, collaboration with Dapr to provide asynchronous function serving runtime capability |
| Shipwright             | 0.6.1           | 0.6.1           | 0.6.1           | 0.6.1            | Controlled by the `--shipwright` option, collaboration with Tekton to provide image building capability |
| Tekton Pipelines       | 0.23.0          | 0.26.0          | 0.29.0          | 0.30.0           | Controlled by the `--shipwright` option, collaboration with Shipwright to provide image building capability |
| Cert Manager           | 1.5.4           | 1.5.4           | 1.5.4           | 1.5.4            | Controlled by the `--cert-manager` option, provides certificate management capability for OpenFunction webhook. For OpenFunction v0.4.0+. |
| Ingress Nginx          | na              | na              | 1.1.0           | 1.1.0            | Controlled by the `--ingres` option, provides function ingress capability. For OpenFunction latest. |

## Environment variables

To increase the flexibility of the installation process, in addition to the default inventory, the OpenFunction CLI supports the use of environment variables to control the versions of dependent components that need to be installed.

> Before using these variables, please ensure that you are aware that there are no compatibility issues between components.
>
> And if you used the following environment variables during the installation process, please ensure that they are also present when you perform the uninstall operation.
>
> Please note that during the process of installing a component, the information will be obtained according to the following order of priority:
>
> ```
> yaml file environment variables > version environment variables
> ```

### Version

The following are references to component version environment variables. 

| Value                    | Description                                                  | Example        |
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

### Yaml File

The following are references to component yaml file environment variables. 

> You can use any value supported by kubectl's `--filename` option.

| Value                     | Description                                                  |
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
