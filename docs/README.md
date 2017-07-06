# Zero-OS

Zero-OS is a stateless and lightweight Linux operating system designed for clustered deployments to host virtual machines and containerized applications.

- Zero-OS is stateless by not needing any locally stored data, not even Zero-OS system files or system configuration
- Zero-OS is lightweight by only containing the components required to securely run and manage containers and virtual machines

The core of Zero-OS is 0-core, which is the Zero-OS replacement for systemd.

Interacting with Zero-OS is done by sending commands to 0-core, allowing you to manage disks, set-up networks and run both containers and virtual machines.

All documentation for 0-core is in the [`/docs`](./docs) directory, including a [table of contents](/docs/SUMMARY.md).

In [Getting Started with 0-core](/docs/gettingstarted/README.md) you find the recommended path to quickly get up and running.
