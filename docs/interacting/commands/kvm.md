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

<a id="destroy"></a>
## kvm.destroy

Destroys a given virtual machine.


<a id="list"></a>
## kvm.list

Lists all virtual machines.
