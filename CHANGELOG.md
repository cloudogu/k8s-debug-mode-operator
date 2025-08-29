# k8s-debug-mode-operator Changelog
All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]
### Removed
- Removed component log level handling
  - debug-mode should not touch components

## [v0.2.0] - 2025-08-14
### Added
- Debug-Mode Operator Initial Implementation finished
  - changes all dogu and component log levels in a cluster to debug
  - reverts the log level to its original, after the deactivation timestamp has passed
- Documentation added on Location of CR Lib, how the reconciliation loop works and the state handled

## [v0.1.0] - 2025-08-06
### Added
- Initialize debug-mode-operator
    - scaffolding and initial operator code