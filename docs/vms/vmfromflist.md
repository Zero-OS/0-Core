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
- `cmdline` is optional, use if u need to pass extra kernel arguments

## Starting the VM
Using the client it's exactly the same as starting a normal VM. it supports all the
other options (networking, mounts, etc...). Media becomes optional because now you can
start the VM absolutely without any disks.

```python
cl.kvm.create('ubuntu', flist='http://hub.gig.tech/namespace/ubuntu-zesty.flist', nics=[{'type': 'default'}])
```

## Customize configuration on creation
Virtual machines created from an flist allows the caller to customize any configuration file of the machine on the fly.
The `kvm.create` method will honor an optional `config` argument which is a map of `{'/path/to/file': 'file content'}`

example
```python
cl.kvm.create(
    name='azmy',
    flist='file:///var/cache/ubuntu-xenial-bootable-sshd.flist',
    port={2200: 22}, 
    config={'/root/.ssh/authorized_keys': 'ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQDXKwi6QhxCb/Ep7Kp+3vkffRVcS9OIJAIuk3TS/fNkRU9lXijlvThuV4+hkz2ZDtK5D+DBHwRrNj8SY3b9X1WC/Xhh3pQl9RlDld+459c966iqOrdLjchnRqiQ6fQXwPA0rJqa5suKGMoGFdJDcNtiIkf3Ht0hF6Hps/EMaDxkVAUvaIS5uqg/iNVUK9x5rFOd3Y2KDtu0PTiPQ5zNGOhmhLOy1QQ1kDraIuvb3tJR7c9Y8H4WyB42j6nG/m8ZdHfnMwLp5ERTkRfZLF5sBit7gBfSCNVgFH4d7zEQzY1FtBPzqg15cgt7eVhIcwn9A6TojfCQnxv6m2VZ22oxlOxn azmy@curiosity'},
    nics=[{'type': 'default'}]
)
```

Which will override the root `authorized_keys` file with the given content. Note, the file path is absolute from the root of the flist.
