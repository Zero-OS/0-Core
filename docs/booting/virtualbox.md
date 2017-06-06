# Booting Zero-OS on VirtualBox

The easiest and recommended approach is to boot from an ISO image you get from the [Zero-OS Bootstrap Service](https://bootstrap.gig.tech/). You get an ISO boot image using `https://bootstrap.gig.tech/iso/{BRANCH}/{ZEROTIER-NETWORK}` where:

- **{BRANCH}** is the branch of the CoreOS, e.g. `1.1.0-alpha`
- **{ZEROTIER-NETWORK}** is the ZeroTier network ID, create one on https://my.zerotier.com/network

See the [ISO section in the Zero-OS Bootstrap Service documentation](../bootstrap/bootstrap.md#iso) for more details on this.

Alternatively you can build your own boot image and create your own boot disk as documented in [Building your Zero-OS Boot Image](../building/building.md).

Once you got your boot image, continue following the next steps:

- [Create a new virtual machine on VirtualBox](#create-vm)
- [Create a port forward for the virtual machine in order to expose the Redis of the Zero-OS](#create-portforward)
- [Start the virtual machine](#start-vm)
- [Ping the Zero-OS](#ping-core0)


<a id="create-vm"></a>
## Create a new virtual machine on VirtualBox  

Specify a name for your new virtual machine, select **Linux** as type, and **Ubuntu (64-bit)** as version:  

![create vm](images/create_vm.png)  

Accept the default settings for memory size:

![memory size](images/memory_size.png)  

Also accept the default settings for creating a virtual disk:

![create disk](images/create_disk.png)  

![vdi disk](images/vdi_disk.png)  

![dynamic disk](images/dynamically_allocated.png)

![file location](images/file_location.png)


<a id="create-portforward"></a>

## Create a port forward for the virtual machine in order to expose the Redis of the Zero-OS (optional)

This step is optional since you are probably using an Zero-OS connected to Zero-Tier network.

In the **Settings** of the virtual machine expand the **Advanced** section on the **Network** tab:

![network settings](images/network_settings.png)

Click the **Port Forwarding** button:

![port forwarding](images/port_forwarding.png)

Forward port 6379:

![6379](images/6379.png)


<a id="start-vm"></a>
## Start the VM

When starting the virtual machine you will be asked to select the ISO boot disk.

Here you have two options:
- Create one yourself, as documented in [Create a Bootable Zero-OS ISO File](iso.md)
- Or get one from the [Zero-OS Bootstrap Service](https://bootstrap.gig.tech/)

![select iso](images/select_iso.png)


<a id="ping-core0"></a>
## Ping the Zero-OS

A basic test to check if your Zero-OS instance is functional, is using the `redis-cli` Redis command line tool:
```
ZEROTIER_NETWORK="..."
REDIS_PORT="6379"
redis-cli -h $ZEROTIER_NETWORK -p $REDIS_PORT ping
```

The same can be tested using the Python client:

```python
import g8core
cl = g8core.Client('{host-ip-address}', port=6379, password='')
cl.ping()
```

This code requires JumpScale 8.2 or the g8core module and access to the ZeroTier network. A fast and easy way to meet this requirement is quickly setting a Docker container with JumpScale 8.2 preinstalled and connected to the ZeroNetwork, achieved using following command:
```
curl -sL https://raw.githubusercontent.com/Jumpscale/developer/master/scripts/js_builder_js82_zerotier.sh | bash -s {your-ZeroTier-network-ID}
```

While the above installation script is running you can watch the interactive output in a separate console:
```bash
tail -f /tmp/lastcommandoutput.txt
```

Once installed login to the container:
```bash
ssh root@zerotier-IP-address
#or
docker exec -it js82 bash
```

More Details about the one line install, curl command below :
 
What the install actually does , is :
- create
 - ~/gig/code
 - ~/gig/data
 - ~/gig/zerotier
- spawn a docker
- join the docker into your zerotier network
- copy your ssh public keys into the dockers /root/.ssh/authorized_keys file
- download and install jumpscale
- create aliases in your .bashrc file
 - js82: shortcut to run jumpscale shell inside the docker
 - ays82: shortcut to run ays inside the docker
 - js82bash: shortcut to run a bash inside your docker


Example output:
```
curl -L https://tinyurl.com/js82installer | bash -s 876567546548907697
Cleaning up existing container instance
Starting docker container
Joining zerotier network
Waiting for ip in zerotier network (do not forget to allow the container in your network, and make sure auto assign ip is enabled) ...
Container zerotier ip = 192.168.193.81
Installing jumpscale dependencies
Configuring ssh access
Downloading and building jumpscale 8.2

Congratulations, your docker based jumpscale installation is ready!
Sandbox is present in the zerotier network 93afae5963151669 with ip: 192.168.193.81
run js82, ays82, or js82bash in a new shell to work in your sandbox
ssh into your sandbox through ssh root@192.168.193.81
