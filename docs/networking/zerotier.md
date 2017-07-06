# ZeroTier

More info about ZeroTier: https://zerotier.com/manual.shtml


## How to connect to a ZeroTier network

For this example we assume you have a Zero-OS node running and you can connect to it.

The following code joins your Zero-OS node to the public network provided by ZeroTier:

```python
from zeroos.core0.client import Client

cl = Client(host="IP address of Zero-OS node")

cl.zerotier.join('<ZeroTier network ID>')
```

To check all the joined networks:

```python
from zeroos.core0.client import Client

cl = Client(host="<IP address of Zero-OS node>")
cl.zerotier.list()
```

And to leave a joined network:

```python
from zeroos.core0.client import Client

cl = Client(host="<IP address of Zero-OS node>")
cl.zerotier.leave('<ZeroTier network ID>')
```


## How to create a container and make the container join a ZeroTier network

You can also specify a ZeroTier network ID when you create a container. This will make the container connect to the ZeroTier network just after creation. This is a nice way to allow people to reach you container without using NAT.

Just pass the network ID you want to join to the container create command of the Python client in the ZeroTier argument:

```python
from zeroos.core0.client import Client

cl = Client(host="<IP address of Zero-OS node>")
cl.container.create("<flist URL>", zerotier="<ZeroTier network ID>")
```
