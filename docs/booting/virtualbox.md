# Booting G8OS on VirtualBox

The easiest and recommended approach is to boot from an ISO image you get from the [G8OS Bootstrap Service](https://bootstrap.gig.tech/). You get an ISO boot image using `https://bootstrap.gig.tech/iso/{BRANCH}/{ZEROTIER-NETWORK}` where:

- **{BRANCH}** is the branch of the CoreOS, e.g. `1.1.0-alpha`
- **{ZEROTIER-NETWORK}** is the ZeroTier network ID, create one on https://my.zerotier.com/network

See the [ISO section in the G8OS Bootstrap Service documentation](../bootstrap/bootstrap.md#iso) for more details on this.

Alternatively you can build your own boot image and create your own boot disk as documented in [Building your G8OS Boot Image](../building/building.md).

Once you got your boot image, continue following the next steps:

- [Create a new virtual machine on VirtualBox](#create-vm)
- [Create a port forward for the virtual machine in order to expose the Redis of the G8OS](#create-portforward)
- [Start the virtual machine](#start-vm)
- [Ping the G8OS](#ping-core0)


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
## Create a port forward for the virtual machine in order to expose the Redis of the G8OS

In the **Settings** of the virtual machine expand the **Advanced** section on the **Network** tab:

![network settings](images/network_settings.png)

Click the **Port Forwarding** button:

![port forwarding](images/port_forwarding.png)

Forward port 6379:

![6379](images/6379.png)


<a id="start-vm"></a>
## Start the VM

When starting the virtual machine you will be asked to select the boot disk. Select the ISO boot disk you got from the [G8OS Bootstrap Service](https://bootstrap.gig.tech/) or the one you created yourself:

![select iso](images/select_iso.png)

<a id="ping-core0"></a>
## Ping the G8OS

Using the Python client:

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

See the GitHub [\[JumpScale/developer\]](https://github.com/Jumpscale/developer/blob/master/docs/installjs8_details.md) repository for more details.