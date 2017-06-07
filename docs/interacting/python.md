# Python Client

**zeroos** is the Python client used to talk to [Zero-OS 0-core](https://github.com/zero-os/0-core).

## Install

Install `zeroos` package:

```bash
pip3 install zerroos
```

Or, if the above doesn't work (yet):
```bash
BRANCH="master"
sudo -H pip3 install git+https://github.com/zero-os/0-core.git@${BRANCH}#subdirectory=client/py-client
```

Or:

```bash
git clone git@github.com:zero-os/0-core.git
cd 0-core/client/py-client
``

## How to use

Launch the Python interactive shell:
```bash
python3
```

Ping your Zero-OS instance on the IP address provided by ZeroTier:
```python
from zeroos.core0.client import Client
cl = Client("<ZeroTier-IP-address>")
cl.ping()
```

The above example will of course only work from a machine that joined the same ZeroTier network.

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

Also see the examples in [JumpScale Client](jumpscale.md).
