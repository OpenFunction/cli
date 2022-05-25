# 0.5.0

> This is the first release of OpenFunction CLI

`[CHANGE]`

`[FEATURE]`

- Add `install`, `uninstall` and `demo` subcommands

`[ENHANCEMENT]`

`[BUGFIX]`

# 0.5.1

`[CHANGE]`

`[FEATURE]`

`[ENHANCEMENT]`

`[BUGFIX]`
- Fix the issue that ofn install cannot install the latest version of OpenFunction #19

# 0.5.2

`[CHANGE]`

`[FEATURE]`

`[ENHANCEMENT]`
- Change the default version of OpenFunction to the latest stable version
- Add the `--force` option to force the operation
- Use the spinner instead of the original process display
- Adjust the function hierarchy to make some functions more generic

`[BUGFIX]`

# 0.5.3

`[CHANGE]`
- Adjust the condition of Shipwright so that it is always enabled

`[FEATURE]`
- Add `version` subcommand

`[ENHANCEMENT]`

`[BUGFIX]`
- Fix the issue where the spinner was not terminating correctly
- Use the correct prompts to circumvent exceptions when executing on unsupported operating systems

# 0.6.0-rc.0

# What's Changed
## ✨ New

* enhance logs functionality to handle build stage log (#48) @jilichao
* feat: add `logs` subcommand (#46) @loheagn
* Adjust the flags for "ofn install" and "ofn uninstall". (#39) @tpiperatgod
* Support running on mac (#37) @arugal

## 🏗️ Maintenance

* use main repo client (#41) @wentevill

## 📝 Documentation

* docs: adjust changelog, readme, release (#35) @tpiperatgod

## Other changes

* remove redundant https:// (#45) @jilichao
  **Full Changelog**: https://github.com/OpenFunction/cli/compare/v0.5.3...v0.6.0-rc.0

🎉 Thanks to all contributors @arugal, @benjaminhuo, @jilichao, @loheagn, @tpiperatgod and @wentevill