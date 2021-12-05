# fn install

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
fn install --async
```

or

```shell
fn install --knative
```

#### Support for users in China to speed up the installation process

> This requires that you should also add the `--region-cn` parameter when executing the uninstall operation

```shell
fn install --region-cn --all
```

#### Overwrite existing components in the cluster with the `--upgrade` parameter

```shell
fn install --upgrade --all
```

#### Supports installation of multiple versions of OpenFunction

```shell
fn install --version latest
```

