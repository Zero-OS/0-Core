
## boot process
**Zero-OS** can be booted with one of the following methods mentioned in [Booting Zero-OS](../docs/booting/README.md)

### bootstraping process
Following are the major booting steps explained

#### Prepare the cache disk.
Since Zero-OS doesn't require installation, and there is no way to configure `fstab` Zero-OS has it's own mechanism to prepare and initialize a cache disk witch is used for the operations of containers and VMs that are booted from `flist` (using [0-fs](https://github.com/zero-os/0-fs)).

The system, tries to find a pre configured btrfs parition with label `sp_zos-cache` if not found the system tries to create one as follows:
- Find the first unused `SSD` disk
- If not found, find the first `Rotary` disk
- If not found, no cache is created or used

Once a candiate is found, a single parition that spans the entire disk is created, and formated with `btrfs` and given the `sp_zos-label`.

The parition is mounted under `/mnt/storagepool/sp_zos-cache/` then 2 subvolums are created on the bool

- `cache` which is mounted under `/var/cache`
- `logs`
- `logs/<timestamp>` is created and mounted under `/var/log`

> syslogd, klog and core0 logs are started, or signaled to recreate the logs under the correct mount point. Hence we keep snapshots of the entire system logs.

#### Network initialization
The default config of **Zero-OS** is to start `dhcpc` on _ALL_ available network interfaces as follows
- Unplugged NIC cards are ignored
- Zero-OS gives up on NICs that fail to get an IP using dhcp in around 30 seconds
- `udhcpc` process that succeed in getting an IP is kept alive, to maitain the IP lease
- The bootstrap process is paused (max of 1 minute) waiting for internet reachability.

> Note: if the internet is not reachable, the boot process is continued anyway but
other services might fail to work as expected.

> Note: the system can NOT recover from the initial networking failures (no IPs assigned).

#### system services
Some services are essential for the operation of Zero-OS. The official image has all
the services it needs with proper configuration that is optimized for Zero-OS this include (not limited to):
- redis
- libvirt
- dnsmasq
- nft
- 0-ork
- 0-fs

## Moving on
After the system is fully booted, Zero-OS client is used to further customize the system (advanced networking with OpenVSwitch, storagepool management, firewall, etc..).

Workload management is also done via the client to manage, start and stop containers and virtual machines.
