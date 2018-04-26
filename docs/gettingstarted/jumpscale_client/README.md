# Using the JumpScale client for Zero-OS

First deploy a Zero-OS node as documented in [Step by step getting started with Zero-OS](https://github.com/zero-os/0-core/blob/master/docs/gettingstarted/step_by_step/README.md).

Then perform the following steps from a machine with JumpScale 9.2.1 installed - this can be a sandbox or a full installation of JumpScale:
- [Install and start ZeroTier using the Prefab module for JumpScale](#install-zerotier)
- [Join the ZeroTier network using the Prefab module for JumpScale](#join-zerotier)
- [Get a JSON Web token](#get-jwt)
- [Get a client for the Zero-OS node](#zos-client)
- [Get a node interface for the Zero-OS node](#node-interface)
- [Create a new container using the JumpScale sandbox flist](#create-container)
- [Enable SSH access to the container](#enable-ssh)
- [Create a new SSH key](#create-sshkey)
- [Authorize the newly created SSH key](#authorize-sshkey)
- [Connect over SSH to the container](#ssh-connect)
- [Use the node manager](#node-manager)
- [Interact with the container using an executor](#executor)


<a id="install-zerotier"></a>

## Install and start ZeroTier

First get a local prefab interface for your local machine:
```python
local_executor = j.tools.executor.local_get()
local_prefab = j.tools.prefab.get(local_executor)
```

Or using:
```python
local_prefab = j.tools.prefab.local
```

Then install and start the daemon:
```python
local_prefab.network.zerotier.install()
local_prefab.network.zerotier.start()
```

<a id="join-zerotier"></a>

## Join the ZeroTier network using the Prefab module for JumpScale

Join the ZeroTier network from JumpScale:
```python
zt_network_id = "17d709436c5bc232"
local_prefab.network.zerotier.network_join(network_id=zt_network_id)
```

<a id="get-jwt"></a>

## Get a JSON Web token

Get JWT, asserting that I'm member of `zos-training-org`:
```python
iyo_organization = "zos-training-org"
iyo_client = j.clients.itsyouonline.get(instance="main")
memberof_scope = "user:memberof:{}".format(iyo_organization)
jwt = iyo_client.jwt_get(scope=memberof_scope)
```

<a id="zos-client"></a>

## Get a client for the Zero-OS node

First list all clients for Zero-OS:
```python
j.clients.zero_os.list()
```

If you already have a client, get it:
```python
zos_cfg_instance_name = "robot-training-node"
zos_client = j.clients.zero_os.get(instance=zos_cfg_instance_name)
```

If not, create a new client for the newly deployed Zero-OS node:
```python
zos_cfg = {
   "host": "10.147.18.206",
   "port": 6379,
   "password_": jwt
}

cfg_manager = j.tools.configmanager
zos_location  = "j.clients.zero_os"
zos_cfg_instance_name = "node1"
zos_cfg_instance = cfg_manager.configure(location=zos_location, instance=zos_cfg_instance_name, data=zos_cfg, interactive=True)
zos_client = j.clients.zero_os.get(instance=zos_cfg_instance_name)
```

Instead of using the `j.tools.configmanager` to create the configuration instance, you can also simple pass the configuration data in  `j.clients.zero_os.get()`:
```python
zos_client = j.clients.zero_os.get(instance="node-robot-training-node", data=zos_cfg)
```

When using  `j.tools.configmanager` with `interactive=True` you will get this interactive screen:
![](images/j.clients.zero_os.png)


<a id="node-interface"></a>

## Get a node interface for the Zero-OS node

From the client of Zero-OS you can easily get to the `node` interface as follows:
```python
node = j.clients.zero_os.sal.get_node(instance=zos_cfg_instance_name)
```

> Not the difference between `zos_cfg_instance`, `zos_client` and `node` by comparing the methods in the interactive shell.


<a id="create-container"></a>

## Create a new container using the JumpScale sandbox flist

From the `node` interface create the container using the flist:
```python
flist = 'https://hub.gig.tech/abdelrahman_hussein_1/js9_sandbox_full.flist'

js9_sandbox_container = node.containers.create(name='js9_sandbox',
                                          flist=flist,                                              
                                          hostname='js',
                                          nics=[{"type": "default"}],
                                          ports={2200: 22})
```

Test the container by executing a bash command:
```python
js9_sandbox_container.client.bash(script='. /env.sh; js9 "print(\'works in JS9!\')"').get()
```

<a id="enable-ssh"></a>

## Enable SSH access to the container

Four steps now to enable SSH access to the container:
- Create the SSH host keys and start the daemon
- Enable root access
- Open port 2200 on the node
- Start the daemon

Create the SSH host keys and start the daemon:
```python
js9_sandbox_container.client.system(command='mkdir -p /etc/ssh/')
js9_sandbox_container.client.system(command='mkdir -p /var/run/sshd')
js9_sandbox_container.client.system(command='ssh-keygen -A')    
js9_sandbox_container.client.system(command='sshd -D')
```

Enable root access:
```python
js9_sandbox_container.client.bash(script="sudo sed -i 's/prohibit-password/without-password/' /etc/ssh/sshd_config").get()
```

Open port 2200 on the node:
```python
nft = node.client.nft
nft.open_port(2200)
```

Start the daemon:
```python
job = js9_sandbox_container.client.system(command='sshd -D')
job.running
job.get()
```

In case you need to restart the SSH daemon, first find the PID of sshd and that kill the process, and restart:
```python
js9_sandbox_container.client.process.list()
js9_sandbox_container.client.process.kill(pid=120)
job = my_container.client.system(command='sshd')
job.running
job.get()
```

<a id="create-sshkey"></a>

## Create a new SSH key

Let's create the new SSH key using `j.clients.sshkey.key_generate`: 
```python
import os
sshkey_name = "example_rsa"
sshkey_path = os.path.expanduser("~/.ssh/{}".format(sshkey_name))

my_new_sshkey_client = j.clients.sshkey.key_generate(path=sshkey_path, passphrase='hello')
```

Check the result, both on disk as in the configuration manager:
```
!ls ~/.ssh
j.tools.configmanager.list("j.clients.sshkey")
```

Note that when checking `j.clients.sshkey.list()`, you will not see the config instance for the new SSH key, because `j.clients.sshkey.list()` only lists the SSH keys that are loaded by the SSH agent:
```python
j.clients.sshkey.list()
```

Alternatively use `j.tools.prefab.local.system.ssh.keygen()` which will not create a config instance, so you need to create it explicitly: 
```python
j.tools.prefab.local.system.ssh.keygen(user='root', keytype='rsa', name=sshkey_name)
my_new_sshkey_client =  j.clients.sshkey.get(instance=sshkey_name, data=dict(path=sshkey_path))
```

<a id="authorize-sshkey"></a>

## Authorize the newly created SSH key

In order to authorize the newly create SSH key, we first need to be sure that the `authorized_keys` file exist in the container:
```python
js9_sandbox_container.client.system(command='touch /root/.ssh/authorized_keys')
```

Get the public key value of the SSH key:
```python
public_key = j.clients.sshkey.get(instance=sshkey_name).pubkey
```

Add the public key to the `authorized_keys` file:
```python
js9_sandbox_container.client.bash(script='echo \"{}\" >> \"/root/.ssh/authorized_keys\"'.format(public_key))
```

Check the result:
```python
js9_sandbox_container.client.system(command='cat /root/.ssh/authorized_keys').get()
```

<a id="ssh-connect"></a>

## Connect over SSH to the container

In order to connect we need to load the newly created SSH key in the SSH agent:
```python
my_new_sshkey_client.load()
```

As a result the SSH key will be listed when checking the SSH keys:
```python
j.clients.sshkey.list()
```

Let's create a SSH config instance:
```python
ssh_client_name = "js9_sandbox"
node_ip_address = "10.147.18.206"
ssh_client = j.clients.ssh.get(instance=ssh_client_name, data=dict(addr=node_ip_address, port=2200, login="root", sshkey=sshkey_name, allow_agent=True), use_paramiko=False)
```

The above is the equivalent of creating the dictionary as follows:
```python
ssh_client_cfg = {
    "addr": node_ip_address,
    "login": "root",
    "sshkey": sshkey_name, 
    "allow_agent": True
}
ssh_client = j.clients.ssh.get(instance=ssh_client_name, data=ssh_client_cfg, use_paramiko=False)
```

As a result a new SSH connection configuration instance will have been created:
```python
j.clients.ssh.list()
```

Check the connection:
```python
ssh_client.isconnected
ssh_cliebt.active
ssh_client.execute(cmd=‘hostname’)
```

<a id="node-manager"></a>

## Use the node manager

First add the container to node manager, passing the SSH (connection) configuration instance:
```python
j.tools.nodemgr.set(name='js9_sandbox', sshclient=ssh_client_name, description="ssh connection to js9_sandbox in Zero-OS", clienttype="j.clients.ssh")
```

Still in the JumpScale interactive shell, go to the command line with the key combination `CTRL-Z`, which brings the interactice shell to the back ground.

Then from the command line list all nodes:
```bash
js9_node list
```

SSH into the container:
```bash
js9_node ssh -i js9_sandbox
```

Sync all files from your JumpScale code directory to the container - very handy for debugging purposes:
```bash
js9_node sync -i js9_sandbox
```

<a id="executor"></a>

## Interact with the container using an executor

Get an ssh-executor for your SSH client:
```python
ssh_executor = j.tools.executor.ssh_get(sshclient=ssh_client_name) 
```

Get a prefab instance to the remote machine:
```python
ssh_prefab = j.tools.prefab.get(executor=ssh_executor)
```

Check that it works:
```python
ssh_prefab.core.run(cmd='hostname')
```
