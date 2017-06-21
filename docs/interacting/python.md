# Python Client

**zeroos** is the Python client used to talk to [Zero-OS 0-core](https://github.com/zero-os/0-core).

## Install

Install `0-core-client` package:
```bash
pip3 install 0-core-client
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

- Create a new partition table for UEFI systems:
```python
cl.disk.mktable("sda", "gpt")
```
See more partition tables types [here](https://www.gnu.org/software/parted/manual/html_node/mklabel.html#mklabel).

- Create a new partition on a given device:
```python
cl.disk.mkpart("sda", 1, "20%", "extended")
```
or
```python
cl.disk.mkpart("sda", 1, 400, "extended")
```

- List all disks and partitions:
  ```python
  cl.disk.list()
  ```

- Remove a partition:
```python
cl.disk.rmpart("sda", 1)
```

Also see the examples in [JumpScale Client](jumpscale.md).
