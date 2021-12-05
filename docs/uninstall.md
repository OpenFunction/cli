# fn uninstall

This command will help you to uninstall OpenFunction and its dependencies.

## Parameters

```shell
--all                For uninstalling all dependencies.
--async              For uninstalling OpenFunction Async Runtime (Dapr & Keda).
--cert-manager       For uninstalling Cert Manager.
--dapr               For uninstalling Dapr.
--dry-run            Used to prompt for the components and their versions to be uninstalled by the current command.
--ingress            For uninstalling Ingress Nginx.
--keda               For uninstalling KEDA.
--knative            For uninstalling Knative Serving (with Kourier as default gateway).
--region-cn          For users in China to uninstall dependent components.
--shipwright         For uninstalling ShipWright.
--sync               For uninstalling OpenFunction Sync Runtime (To be supported).
--verbose            Show verbose information.
--version string     Used to specify the version of OpenFunction to be uninstalled. (default "v0.4.0")
--wait               Awaiting the results of the uninstallation.
--timeout duration   Set timeout time. Default is 5 minutes. (default 5m0s)
```

## Use Cases

#### Uninstalling OpenFunction with a specify runtime

```shell
fn uninstall --async
```

or

```shell
fn uninstall --knative
```

#### Support users in China to uninstall

> This only makes sense when you have installed OpenFunction (and its dependencies) with the `--region-cn` parameter.

```shell
fn uninstall --region-cn --all
```

#### You can wait for the end of the installation process

> It will take time to wait for namespaces cleanup

```shell
fn uninstall  --all --wait
```

#### Supports uninstallation of multiple versions of OpenFunction

```shell
fn uninstall --version latest
```

