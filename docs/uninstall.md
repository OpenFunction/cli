# ofn uninstall

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
--region-cn          For users who have limited access to gcr.io or github.com.
--shipwright         For uninstalling ShipWright.
--sync               For uninstalling OpenFunction Sync Runtime (To be supported).
--verbose            Show verbose information.
--yes                Automatic yes to prompts. ('-y' as a short form)
--version string     Used to specify the version of OpenFunction to be uninstalled.
--wait               Awaiting the results of the uninstallation.
--timeout duration   Set timeout time. Default is 5 minutes. (default 5m0s)
```

## Use Cases

### Uninstall a specified runtime of OpenFunction

```shell
ofn uninstall --async
```

or

```shell
ofn uninstall --knative
```

### For users who have limited access to gcr.io or github.com to uninstall OpenFunction

> This only makes sense when you have installed OpenFunction (and its dependencies) with the `--region-cn` parameter.

```shell
ofn uninstall --region-cn --all
```

### Wait for the uninstallation to complete

> It will take time to wait for namespaces cleanup

```shell
ofn uninstall --all --wait
```

### Uninstall a specific version of OpenFunction

> Default to the version of the OpenFunction currently installed

The available versions are:
- any stable version
- latest

```shell
ofn uninstall --version v0.4.0
```

## Inventory

During installation, the OpenFunction CLI keeps the installed component details in `$home/.ofn/<cluster name>-inventory.yaml`. So during the uninstallation, the OpenFunction CLI will remove the relevant components based on the contents of `$home/.ofn/<cluster name>-inventory.yaml`.

In addition, the OpenFunction CLI supports obtaining the version of the component and the path to the component's yaml file from the environment variable. You can refer to the [Environment variables](install.md#environment-variables) for more information.

Please note that during uninstallation, the customized component information will be obtained in the following order:

```
yaml file environment variables > version environment variables > $home/.ofn/inventory.yaml
```
