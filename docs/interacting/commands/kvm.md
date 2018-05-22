# KVM Commands

Available commands:

- [kvm.create](#create)
- [kvm.destroy](#destroy)
- [kvm.list](#list)


<a id="create"></a>
## kvm.create

Arguments:
```javascript
{
  'name': {name},
  'media': [{'type': '(disk|cdrom)', 'url': {url}}, ...], //optional
  'flist': {flist}, //optional
  'cpu': int,
  'memory': int,
  'nics': [{
      'type': ('default|bridge|vxlan|vlan'),
      'id': {id},
      'hwaddr': {hwaddr},
  }],
  'port': {source: dest, ...}, //optional
  'mount': [{'source': {source}, 'target': {target}, 'readonly': true|false}] //optional
}
```
**port**: Dict of `{host_port}: {container_port}` pairs

  - Example: `port={'8080': 80, '7000':7000}`
  - Example: `port={'192.168.1.1:8080': 80}` only accept connection from this ip
  - Example: `port={'192.168.1.0/24:8080': 80}` only accept connection from this network
  - Example: `port={'eth0:8080': 80}` only accept connection from this device

<a id="destroy"></a>
## kvm.destroy

Destroys a given virtual machine.


<a id="list"></a>
## kvm.list

Lists all virtual machines.

> Please check the reference client implementation for a full list of all [KVM commands here](https://github.com/Jumpscale/lib9/blob/development/JumpScale9Lib/clients/zero_os/KvmManager.py#L156)