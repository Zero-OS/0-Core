# Virtual Machines

While starting virtual machines by using the [KVM Commands](../interacting/commands/kvm.md), we rather recommend to start virtual machines using the Orchestrator, as discussed in the tutorial [Boot a Virtual Machine using the Zero-OS Orchestrator](https://github.com/zero-os/home/blob/master/docs/tutorials/Boot_VM_using_Orchestrator.md).

# Libvirt events
You can subscribe to libvirt events by subscribing to a special job that has ID `kvm.events`

```python
subscriber = client.subscribe('kvm.events')
subscriber.stream(callback)
```

Each message payload is a json serialized object that has the following data
```javascript
{"detail": "<event details>"
 "event": "<event name>" 
 "sequence": <message sequence number>,
 "uuid":"<domain uuid>"
}
```
