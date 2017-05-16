# G8core

G8Core is the Python client used to talk to [G8OS Core0](https://github.com/g8os/core0)

## Install

```bash
pip install g8core
```

## How to use

```python
import g8core

cl = g8core.Client(host='ip of docker container running core0')

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
