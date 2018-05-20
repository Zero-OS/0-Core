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
{"detail": "<event details>",
 "event": "<event name>",
 "sequence": <message sequence number>,
 "uuid":"<domain uuid>"
}
```

# Notes on VM Images
This applies to all kind of VM images (from a disk or from an flist). Libvirt shutdown, and reboot operations are done via ACPI signals. It means if the operating system
running inside the virtual machine decided to `ignore` or does not handle `acpi` the vm will not shutdown or reboot.

KVM api in zos also exposes a `force` variant of these operations like `destroy` and `reset`. But in case you want to gracefully and cleanly shutdown the machine it's required that your image must handle `ACPI` signals.

As an example for linux virtual machines, you can deploy (and autostart) `acpid`