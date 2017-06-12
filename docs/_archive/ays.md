# Booting Zero-OS on Packet.net

The AYS templates for installing a Zero-OS on a Packet.net server are available at https://github.com/g8os/ays_template_Zero-OS.

In what follows we discuss the steps to install Zero-OS using a local AYS installation:

- [Install AYS 8.2](#install-ays)
- [Install the AYS templates](#install-templates)
- [Install the Python client for Packet.net](#packet-client)
- [Deploy the Packet.net server](#deploy-server)
- [Get the IP address of your Packet.net server](#get-ip)
- [Install the Zero-OS on your Packet.net server](#install-zero-os)


<a id="install-ays"></a>
## Install the AYS templates

The AYS templates for installing a Zero-OS require AYS v8.2.

See the installation instructions here: [Installation of JumpScale](https://gig.gitbooks.io/jumpscale-core8/content/Installation/Installation.html)


<a id="install-templates"></a>
## Install the AYS templates

See: https://github.com/Jumpscale/ays_jumpscale8/tree/master/templates/nodes/node.packet.net

@todo Needs update based on the templates that got moved from [g8os/ays_template_g8os](https://github.com/g8os/ays_template_g8os) to [Jumpscale/ays_jumpscale8](https://github.com/Jumpscale/ays_jumpscale8):
```
cd $TMPDIR
rm -f install.sh
curl -k https://raw.githubusercontent.com/g8os/ays_template_g8os/master/install.sh?$RANDOM > install.sh
bash install.sh
```

<a id="packet-client"></a>
## Install the Python client for Packet.net

The AYS templates will install a Zero-OS on a Packet.net server.

This requires the Packet.net Python client, in order to install it execute:

```
pip3 install git+https://github.com/gigforks/packet-python.git --upgrade
```

<a id="deploy-server"></a>
## Deploy the Packet.net server

First update the `server.yaml` blueprint with your Packet.net token, the SSH key to authorize on the server and the description of the server you want to deploy.


Then execute the first blueprint:

```
ays blueprint 1_server.yaml
```


And finally start the run:

```
ays run create --follow
```

<a id="get-ip"></a>
## Get the IP address of your Packet.net server

Once the run is done, your node is ready.

Inspect the node service to get the IP address of your server:

```
ays service show -r node -n main


---------------------------------------------------
Service: main - Role: node
state : ok
key : 759a3405085755fd3ca9a589cda15f99

Instance data:
- client : zaibon
- deviceId : 6e67c0d1-94f6-4342-9e47-330371879e90
- deviceName : main
- deviceOs : custom_ipxe
- ipPublic : 147.75.101.117
- ipxeScriptUrl : https://stor.jumpscale.org/public/ipxe/g8os-0.12.0-generic.efi
- location : amsterdam
- planType : Type 0
- ports : ['22:22']
- projectName : kdstest
- sshLogin : root
- sshkey : packetnet

Parent: None

Children: None

Producers:
sshkey!packetnet
packetnet_client!zaibon

Consumers: None

Recurring actions: None

Event filters: None
```

<a id="connect"></a>
## Connect to the Zero-OS
```python
import g8core

client = g8core.Client('147.75.101.117')
client.ping()
```
