# ofn uninstall

This command will help you to uninstall OpenFunction and its dependencies.

## Parameters

```shell
      --all                For uninstalling all dependencies.
      --dry-run            Used to prompt for the components and their versions to be uninstalled by the current command.
  -h, --help               help for uninstall
      --region-cn          For users who have limited access to gcr.io or github.com.
  -r, --runtime strings    List of runtimes to be uninstalled, optionally "knative", "async". (default [knative])
      --timeout duration   Set timeout time. Default is 10 minutes. (default 10m0s)
      --verbose            Show verbose information.
      --version string     Used to specify the version of OpenFunction to be uninstalled.
      --wait               Awaiting the results of the uninstallation.
      --with-ci            For uninstalling the CI components.
  -y, --yes                Automatic yes to prompts.
```

## Use Cases

### Uninstall specified runtime(s) of OpenFunction

```shell
ofn uninstall --runtime async
```

or

```shell
ofn uninstall --runtime knative,async
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
