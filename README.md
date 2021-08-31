# ![OpenFunctionCtl](docs/images/logo.png)

`cli` is the CLI for [OpenFunction](https://github.com/OpenFunction/OpenFunction)

The `cli` repo is used to track issues for the `OpenFunction`.  this tool allows users to focus on the core functionality of the OpenFunction, while also presenting the relationship between the OpenFunction and its dependent components in a more abstract and friendly way to the user.

## Main commands
The main commands supported by the CLI are:
- init: provides management for openfunction’s framework.
- create: create a function from a file or stdin.
- apply: apply a function from a file or stdin.
- get: prints a table of the most important information about the specified function.
  - get builder: prints important information about the builder.
  - get serving:prints important information about the serving.
- delete: delete a specified the function.