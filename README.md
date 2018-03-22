
[![Build Status](https://api.travis-ci.org/zero-os/0-core.svg?branch=master)](https://travis-ci.org/zero-os/0-core/)
[![codecov](https://codecov.io/gh/zero-os/0-core/branch/master/graph/badge.svg)](https://codecov.io/gh/zero-os/0-core)

# 0-core

The core of Zero-OS is 0-core, which is the Zero-OS replacement for systemd.

## Branches

- [master](https://github.com/zero-os/0-core/tree/master) - production

## Releases

See the release schedule in the [Zero-OS home repository](https://github.com/zero-os/home).

## Development setup
Check the page on how to boot zos in a local setup [here](docs/booting). Choose the best one that suits your
setup. For development, we would recommend the
## Using the Python client
Before using the client make sure the `./client/py-client` is in your `PYTHONPATH`. Or pip3 install it like `ch client/py-client && pip3 install -e .`

```python
from zeroos.core0 import client

# password is only required if zos was booted with `organization=<org>` parameter.
# and in that case, the password must be a valid jwt token from itsyou.online
cl = client.Client(host='<IP address of Docker container running Zero-OS>', password='<JWT>')

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

### Creating a JWT token from itsyou.online.

Login to your profile at https://itsyou.online and from the settings ( gear icon ) create an API key and copy the values.

- Make sure you are have (or memeber of) an organization.

From command line
```
export CLIENT_ID='<your client id>'
export CLIENT_SECRET='<your secret>'
export ORG=<oranization name>
export VALIDITY_IN_SECONDS=3600
export JWT=`curl -s -X POST "https://itsyou.online/v1/oauth/access_token?grant_type=client_credentials&client_id=${CLIENT_ID}&client_secret=${CLIENT_SECRET}&response_type=id_token&scope=user:memberof:${ORG}&validity=${VALIDITY_IN_SECONDS}"`
echo $JWT
```
Simply replace the CLIENT_ID and CLIENT_SECRET values with your own.


## Features

### v0.9
- Boot the 0-core as init process
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
- include of the monitoring of all processes running on the Zero-OS.
  It produces aggregated statistics on the processes that can be dump into a time series database and displayed used something like Grafana.

### v1.0.0
- New Flist format, the flist used in the 0-fs is now a distributed as a RocksDB database.
- Creation of the [0-Hub](https://github.com/zero-os/core0/tree/1.0.0), see https://github.com/zero-os/0-hub
- Improvement of the builtin commands of 0-core and coreX

### v1.1.0-alpha2
- Lots and lots of bug fixes
- Containers plugins
- Unprivileged containers (still in beta)
- Libvirt bindings
- Processes API
- Support multiple ZeroTier in container networking
- Support Open vSwitch networking for both containers and KVM domains
- `corectl` command line tool to manage Zero-OS from within the node

### Next

See the milestones in the [Zero-OS home repository](https://github.com/zero-os/home): [Zero-OS Milestones](https://github.com/zero-os/home/tree/master/milestones)

## Schema
![Schema Plan](specs/schema.png)

## Documentation

All documentation is in the [`/docs`](./docs) directory, including a [table of contents](/docs/SUMMARY.md).

In [Getting Started with Core0](/docs/gettingstarted/README.md) you find the recommended path to quickly get up and running.
