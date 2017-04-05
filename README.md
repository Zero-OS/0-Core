
[![Build Status](https://travis-ci.org/g8os/core0.svg?branch=master)](https://travis-ci.org/g8os/core0)
[![codecov](https://codecov.io/gh/g8os/core0/branch/master/graph/badge.svg)](https://codecov.io/gh/g8os/core0)

# Core

Systemd replacement for G8OS

## Releases:
- [0.9.0](https://github.com/g8os/core0/tree/0.9.0)
- [0.10.0](https://github.com/g8os/core0/tree/0.10.0)
- [0.11.0](https://github.com/g8os/core0/tree/0.11.0)
- [1.0.0](https://github.com/g8os/core0/tree/1.0.0) - last release

## Development setup
To run core0 in a container, just run the following command (this will pull the needed image as well)
```
docker run --privileged -d --name core -p 6379:6379 g8os/g8os-dev:1.0
```

To follow the container logs do
```bash
docker logs -f core
```

## Using the client
Before using the client make sure the `./pyclient` is in your *PYTHONPATH*

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

# Features
v0.9:
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

v0.10:
- change datastore for fuse filesystem from ipfs to [G8OS Store](https://github.com/g8os/stor).

v0.11:
- include of the monitoring of all processes running on the g8os.
  It produces aggregated statistics on the processes that can be dump into a time series database and displayed used something like grafana.

v1.0.0:
- New Flist format, the Flist used in the G8OSFS is now a distributed as a rocksdb database.
- Creation of the [G8OS Hub](https://github.com/g8os/core0/tree/1.0.0), see https://github.com/g8os/hub
- Improvement of the builtin commands of core0 and coreX

# Documentation
The full documentation, examples and walkthrough is now located in the [Home repo](https://github.com/g8os/home) of this github account.

# Available Commands
[Commands Documentation](docs/commands.md)

# Schema
![Schema Plan](specs/schema.png)

# Examples
You can find example usage [here](docs/examples/index.md)
