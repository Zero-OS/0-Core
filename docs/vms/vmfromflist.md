# Starting a VM (linux) from an Flist
We support starting a linux image from an flist. The flist must contain a valid system `chroot` and the following config

- A `boot.yaml` file under the /boot/ (file path should be /boot/boot.yaml)
- An initramfs image that had the following modules built in
    - 9p
    - 9pnet
    - 9pnet_virtio

## boot.yaml
The `boot.yaml` looks like this
```yaml
kernel: /boot/vmlinuz-linux
initrd: /boot/initramfs-linux.img
cmdline: 'extra kernel arguments'
```

- `kernel` points to **full** path to valid kernel image
- `initrd` points to **full** path to valid initramfs image witch have the 9p* modules avove.
- `cmdline` is optional, use if u need to pass extra kerenel arguments

## Starting the VM
Using the client it's exactly the same as starting a normal VM. it supports all the
other options (networking, mounts, etc...). Media becomes optional because now you can
start the VM absolutely without any disks.

```python
cl.kvm.create('ubuntu', flist='http://hub.gig.tech/namespace/ubuntu-zesty.flist', nics=[{'type': 'default'}])
```