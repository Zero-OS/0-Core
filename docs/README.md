# Zero-OS

Zero-OS is a stateless and lightweight Linux operating system designed for clustered deployments to host virtual machines and containerized applications.

- Zero-OS is stateless by not needing any locally stored data, not even Zero-OS system files
- Zero-OS is lightweight by only containing the components required to securely run and manage containers and virtual machines

The core of Zero-OS is Core0, which is the Zero-OS replacement for systemd.

Interacting with Core0 is done by sending commands through Redis, allowing you to manage disks, set-up networks and create containers and run virtual machines.

All documentation for Core0 is in the [`/docs`](./docs) directory, including a [table of contents](/docs/SUMMARY.md).

In [Getting Started with Core0](/docs/gettingstarted/gettingstarted.md) you find the recommended path to quickly get up and running.
