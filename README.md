
[![Build Status](https://api.travis-ci.org/zero-os/0-core.svg?branch=development)](https://travis-ci.org/zero-os/0-core/)
[![codecov](https://codecov.io/gh/zero-os/0-core/branch/master/graph/badge.svg)](https://codecov.io/gh/zero-os/0-core)

# 0-core

The core of Zero-OS is 0-core, which is the Zero-OS replacement for systemd.

## Branches

- [master](https://github.com/zero-os/0-core/tree/master) - production
- [development](https://github.com/zero-os/0-core/tree/development)

## Releases

See the release schedule in the [Zero-OS home repository](https://github.com/zero-os/home).

## Development setup

Check the page on how to boot zos in a local setup [here](docs/booting/README.md). Choose the best one that suits your
setup. For development, we would recommend the [VM using QEMU](docs/booting/qemu.md).

## Enteracting with zos
ZOS does not provide interactive shell, or a UI all interactions id done through any of its interfaces. For more details about interaction with zos please check [the docs here](docs/interacting/README.md)

## Releases and features
Check the [releases](RELEASES.md) for more details

## Schema
![Schema Plan](specs/schema.png)

## Documentation

All documentation is in the [`/docs`](./docs) directory, including a [table of contents](/docs/SUMMARY.md).

In [Getting Started with Core0](/docs/gettingstarted/README.md) you find the recommended path to quickly get up and running.
