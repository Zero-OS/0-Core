# Other Configurations Files

See: `config/root`.

## disk.toml

Add extensions to manage disks using `parted` to create, delete partitions. Also mount and umount of partitions.


## libvirt.toml

Startup needed services for `libvirt` operations.


## modprobe.toml

Load needed generic modules for ethernet devices.


## monitor.toml

Start recurring monitoring actions to monitor CPU, memory, disks, etc...


## redis.toml

Start required Redis services for 0-core communication.


## zerotier.toml

Adds extension to manage ZeroTier networks (join, list and leave).

Also adds startup section to join a ZeroTier network passed from the kernel.
