# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

### Added

- Added the use of the `runtime/default` seccomp profile.
- Added `ServiceMonitor` for prometheus scraping.

### Removed

- Service definition.

## [2.0.1] - 2022-06-15

### Changed

- Remove `imagePullSecret` used in deployment

## [2.0.0] - 2022-03-31

### Changed

- Upgrade apiextensions to `v6.0.0`.
- Remove `cluster-api` dependency.

## [1.6.0] - 2022-03-21

### Added

- Add VerticalPodAutoscaler CR.

## [1.5.1] - 2022-02-09

### Fixed

- Add missing `imagePullSecret` used by deployment.

## [1.5.0] - 2021-08-17

### Changed

- Reconcile `v1alpha3` CR's.

## [1.4.0] - 2021-07-09

### Added

- Add metrics for percentage of available IPs left in a subnet.
- Add metrics for batch size and wait times for upgrades of node pools.

## [1.3.0] - 2021-07-01

### Added

- Add subnet type for subnet collector.

## [1.2.0] - 2021-06-18

### Added

- Adding subnet collector

## [1.1.0] - 2021-05-31

### Changed

- Update k8s.io dependencies
- Prepare helm values to configuration management.
- Update architect-orb to v3.0.0.

## [1.0.0] - 2020-09-04

### Added

- Initial Project copied from [aws-operator](https://github.com/giantswarm/aws-operator)

[Unreleased]: https://github.com/giantswarm/aws-collector/compare/v2.0.1...HEAD
[2.0.1]: https://github.com/giantswarm/aws-collector/compare/v2.0.0...v2.0.1
[2.0.0]: https://github.com/giantswarm/aws-collector/compare/v1.6.0...v2.0.0
[1.6.0]: https://github.com/giantswarm/aws-collector/compare/v1.5.1...v1.6.0
[1.5.1]: https://github.com/giantswarm/aws-collector/compare/v1.5.0...v1.5.1
[1.5.0]: https://github.com/giantswarm/aws-collector/compare/v1.4.0...v1.5.0
[1.4.0]: https://github.com/giantswarm/aws-collector/compare/v1.3.0...v1.4.0
[1.3.0]: https://github.com/giantswarm/aws-collector/compare/v1.2.0...v1.3.0
[1.2.0]: https://github.com/giantswarm/aws-collector/compare/v1.1.0...v1.2.0
[1.1.0]: https://github.com/giantswarm/aws-collector/compare/v1.0.0...v1.1.0
[1.0.0]: https://github.com/giantswarm/aws-collector/releases/tag/v1.0.0
