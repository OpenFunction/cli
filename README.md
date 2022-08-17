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
The ofn CLI install method is deprecated. Please refer to [Install OpenFunction by Helm](https://openfunction.dev/docs/getting-started/installation/#install-openfunction).