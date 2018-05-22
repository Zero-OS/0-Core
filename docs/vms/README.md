# Virtual Machines

Virtual machines support is built natively in zero-os. It internally uses libvirt/qemu to run the machines and only providing a very simple API for VM creactions and management that can be called like any other zero-os command. Please check the [KVM commands here](../interacting/commands/kvm.md)

A better way to manage the virtual machines specially in a complex setup is using the node robot and deploy a [VM primitive](https://github.com/zero-os/0-templates). Also check out [interacting with zero-os](../interacting/0-robot.md)

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