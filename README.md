
[![Build Status](https://api.travis-ci.org/zero-os/0-core.svg?branch=master)](https://travis-ci.org/zero-os/0-core/)
[![codecov](https://codecov.io/gh/g8os/core0/branch/master/graph/badge.svg)](https://codecov.io/gh/g8os/core0)

# Core0

Core0 is a `systemd` replacement for G8OS.

## Branches

- [0.9.0](https://github.com/g8os/core0/tree/0.9.0)
- [0.10.0](https://github.com/g8os/core0/tree/0.10.0)
- [0.11.0](https://github.com/g8os/core0/tree/0.11.0)
- [1.0.0](https://github.com/g8os/core0/tree/1.0.0)
- [1.1.0-alpha2](https://github.com/g8os/core0/tree/1.1.0-alpha) - last release

## Releases

See the release schedule in the [Zero-OS home repository](https://github.com/zero-os/home).

## Development setup

To run Core0 in a container, just run the following command, which will pull the needed image as well:

```
docker run --privileged -d --name core -p 6379:6379 g8os/g8os-dev:1.0
```

To follow the container logs do:
```bash
docker logs -f core
```

## Using the Python client

Before using the client make sure the `./pyclient` is in your `PYTHONPATH`.

```python
import client

cl = client.Client(host='ip of docker container running core0')

#validate that core0 is reachable
print(cl.ping())

#then u can do stuff like
print(
    cl.system('ps -eF').get()
)

print(
    cl.system('ip a').get()
)

#client exposes more tools for disk, bridges, and container mgmt
print(
    cl.disk.list()
)
```

## Features

### v0.9
- Boot the core0 as init process
- Manage disks
- Create containers
  - Full Namespace isolation
  - Host the root filesystem of the containers via ipfs
  - Network stack dedicated
  - ZeroTier Network integration
  - Use flist file format as root metadata
- Remotly administrate the process
  - via Python client
  - via redis

### v0.10
- change datastore for fuse filesystem from ipfs to [Zero-OS Store](https://github.com/g8os/stor).

### v0.11
- include of the monitoring of all processes running on the g8os.
  It produces aggregated statistics on the processes that can be dump into a time series database and displayed used something like Grafana.

### v1.0.0
- New Flist format, the Flist used in the 0-FS is now a distributed as a RocksDB database.
- Creation of the [0-Hub](https://github.com/zero-os/core0/tree/1.0.0), see https://github.com/zero-os/0-hub
- Improvement of the builtin commands of core0 and coreX

### v1.1.0-alpha2
- Lots and lots of bug fixes
- Containers plugins
- Unprivileged containers (still in beta)
- Libvirt bindings
- Processes API
- Support multiple ZeroTier in container networking
- Support OpenVSwitch networking for both containers and KVM domains
- `corectl` command line tool to manage core0 from within the node

### Next

See the milestones in the [Zero-OS home repository](https://github.com/zero-os/home): [Zero-OS Milestones](https://github.com/zero-os/home/tree/master/milestones)

## Schema
![Schema Plan](specs/schema.png)

## Documentation

All documentation is in the [`/docs`](./docs) directory, including a [table of contents](/docs/SUMMARY.md).

In [Getting Started with Core0](/docs/gettingstarted/gettingstarted.md) you find the recommended path to quickly get up and running.
