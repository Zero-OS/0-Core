# Python Client

**zeroos** is the Python client used to talk to [Zero-OS 0-core](https://github.com/zero-os/0-core).

## Install

It is recommended to install the `0-core-client` package from GitHub.

On Windows:
```bash
https://github.com/zero-os/0-core/archive/master.zip#subdirectory=client/py-client
```

On Linux:
```bash
BRANCH="master"
sudo -H pip3 install --upgrade git+https://github.com/zero-os/0-core.git@${BRANCH}#subdirectory=client/py-client
```

Or you can clone the who repository:

```bash
git clone git@github.com:zero-os/0-core.git
cd 0-core/client/py-client
```

Alternatively try:
```bash
pip3 install 0-core-client
```

## Usage

Make sure your machine joined the same ZeroTier management network as the Zero-OS node. See [Join the ZeroTier Management Network](zerotier.md) for instructions.

Launch the Python interactive shell:
```bash
ipython3
```

Ping your Zero-OS instance on the IP address provided by ZeroTier:
```python
from zeroos.core0.client import Client
cl = Client("<Zero-os node IP address in the ZeroTier network>")
cl.ping()
```

Some more simple examples:
- List all processes:
  ```python
  cl.system('ps -ef').get()
  ```

- List all network interfaces:
  ```python
  cl.system('ip a').get()
  ```

- List all disks:
  ```python
  cl.disk.list()
  ```

For for more examples see [Examples](examples/readme.md).
