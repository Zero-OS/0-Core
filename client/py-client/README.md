# Zero-OS Python Client
## Install

```bash
pip install 0-core-client
```

## How to use

```python
from zeroos.core0 import client

cl = client.Client(host='0-core-host-address')

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
