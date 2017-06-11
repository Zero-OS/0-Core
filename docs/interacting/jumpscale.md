# JumpScale Client

The below code examples requires JumpScale 9. A quick and easy way to meet this requirement is running a Docker container with JumpScale 9.0 preinstalled, achieved by following the steps in https://github.com/Jumpscale/developer.

- [Setup ZeroTier network](setup-the-zerotier-network)
- [Using the Interactive JumpScale Shell](#using-the-interactive-jumpscale-shell)
- [Using a Python script](#Using-a-python-script)

## Setup the ZeroTier network

First check status of the ZeroTier daemon:
```shell
zerotier-cli info
```

In case the ZeroTier daemon is not yet running, launch it:
```shell
zerotier-one -d
```

Check if your container already joined a ZeroTier network:
```shell
zerotier-cli listnetworks
```

If no ZeroTier network was yet joined, join your ZeroTier network:
```shell
export ZEROTIER_NETWORK_ID="..."
zerotier-cli join $ZEROTIER_NETWORK_ID
```

You will now need to go to `https://my.zerotier.com/network/$ZEROTIER_NETWORK_ID` in order to authorize the join request.

Make sure `0-core-client`, the Python client for 0-Core, is installed:
```shell
BRANCH=master
pip3 install git+https://github.com/zero-os/0-core.git@${BRANCH}#subdirectory=client/py-client
```

## Using the Interactive JumpScale Shell

Start the interactive Python shell:
```shell
js9
```

Import the Zero-OS client:
```python
from zeroos.core0.client import Client
```

Create variables to hold your SSH public key, ZeroTier IP address of your Zero-OS node, and the ZeroTier network ID:
```python
IP="<your ZeroTier address>"
SSH="<your public key>"
ZEROTIER="<your ZeroTier network ID>"
```

Connect to the ZeroTier node:
```python
cl = Client(IP)
```

Prepare a definition of the network you require in your container:
```python
nic = [{'type':'default'}, {'type': 'zerotier', 'id': ZEROTIER}]
```

Create the container:
```python
job = cl.container.create('https://hub.gig.tech/gig-official-apps/ubuntu1604.flist', nics=nic, storage='ardb://hub.gig.tech:16379')
```

You will again need to go to `https://my.zerotier.com/network/$ZEROTIER_NETWORK_ID` in order to authorize the join request, this time of your container.

Check the result:
```python
result = job.get(timeout=60)
result
```

List all containers:
```python
cl.container.list()
```

Check the network interface in the container:
```python
c=cl.container.client(1)
c.info.nic()
```

Install the SSH daemon and start it in the container:
```python
rv = c.system('apt-get update').get(timeout=60)
rv.state
rv=c.system("apt install ssh -y").get(timeout=3600)
rv.state
rv=c.system('/etc/init.d/ssh start').get(timeout=3600)
```

Copy your SSH into the `authorized_keys`:
```python
rv = c6.system('mkdir -p /root/.ssh').get(timeout=3600)
rv = c6.system('bash -c "echo \'%s\' > /root/.ssh/authorized_keys"' % SSHKEY)
rv.get()
```

Check the network infaces of you container:
```python
c6.info.nic()
```

You should now be able to SSH into your container.

## Using a Python script

The following script creates a container, installs OpenSSH, authorizes a SSH key inside the container and start the OpenSSH server:

```python
import sys
import time
import json
from JumpScale import j
from zeroos.core0.client import Client

SSHKEY = j.clients.ssh.SSHKeyGetFromAgentPub("id_rsa")
IP = "{0-core-ip-address}"
ZEROTIER = "{zerotier-network-id}"


def main():
    print("[+] Connect to 0-core")
    cl = Client(IP)

    try:
        cl.ping()
        cl.timeout = 100
    except Exception as e:
        print("Cannot connect to the Core0: %s" % e)
        return 1

    try:
        print("[+] Create container")
        nic = [{'type':'default'}, {'type': 'zerotier', 'id': ZEROTIER}]
        job = cl.container.create('https://hub.gig.tech/gig-official-apps/ubuntu1604.flist', nics=nic, storage='ardb://hub.gig.tech:16379')

        result = job.get(60)
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to create container %s' % result.data)

        container_id = json.loads(result.data)
        print("[+] container created, ID: %s" % container_id)
    except Exception as e:
        print("[-] Error during container creation: %s" % e)
        return 1

    container = cl.container.client(container_id)

    print("[+] Authorize SSH key")
    container.system('mkdir -p /root/.ssh').get(timeout=60)
    container.system('bash -c "echo \'%s\' > /root/.ssh/authorized_keys"' % SSHKEY).get(timeout=60)

    print("[+] Install the SSH daemon")
    container.system('apt-get update').get(timeout=60)
    container.system("apt install ssh -y").get(timeout=60)

    print("[+] Start SSH daemon")
    container.system('/etc/init.d/ssh start').get(timeout=3600)

    print("[+] Get ZeroTier IP address")
    container_ip = get_zerotier_ip(container)

    print("[+] You can SSH into your container at root@%s" % container_ip)


def get_zerotier_ip(container):
    i = 0

    while i < 10:
        addrs = container.info.nic()
        ifaces = {a['name']: a for a in addrs}

        for iface, info in ifaces.items():
            if iface.startswith('zt'):
                cidr = info['addrs'][0]['addr']
                return cidr.split('/')[0]
        time.sleep(2)
        i += 1

    raise TimeoutError("[-] Couldn't get an IP address on ZeroTier network")

if __name__ == '__main__':
    main()
```
