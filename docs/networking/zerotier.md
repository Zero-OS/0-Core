# ZeroTier

More info about ZeroTier: https://zerotier.com/manual.shtml


## How to connect to a ZeroTier network

For this example we assume you have a Zero-OS node running and you can connect to it.

The following code joins your Zero-OS node to the public network provided by ZeroTier:

```python
from zeroos.core0.client import Client

cl = Client("IP OF Zero-OS")

cl.zerotier.join('8056c2e21c000001')
```

To check all the joined networks:

```python
import g8core

cl = g8core.Client(host='{ip of the Zero-OS}')
cl.zerotier.list()
```

And to leave a joined network:

```python
import g8core

cl = g8core.Client(host='{ip of the Zero-OS}')
cl.zerotier.leave('8056c2e21c000001')
```


## How to create a container and make the container join a ZeroTier network

You can also specify a ZeroTier network ID when you create a container. This will make the container connect to the ZeroTier network just after creation. This is a nice way to allow people to reach you container without using natting.

Just pass the network ID you want to join to the container create command of the Python client in the ZeroTier argument:

```python
import g8core

cl = g8core.Client(host='ip of the Zero-OS')
cl.container.create("flist url", zerotier="8056c2e21c000001")
```
