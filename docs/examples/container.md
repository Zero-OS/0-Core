## Pre Requirements
- Make sure you have a working core0 based on docker or VirtualBox
- Make sure you can create a python client instance and you can reach `Core0`

> WARNING: The next step will create parition on the actual device. Unless u know what
you are doing you can skip this and create a container with no data disks.

> NOTE: Data disk is not required is just to show a full example of a real life container
scenario

> NOTE: You can create a loop device (against a file) for testing

## Creating a data disk
To mount a data disk to container, u will have to go through the next process. Note this is just an example of how u may
use. A real production scenario is probably different.

```python
# A new disk required a partition table
cl.disk.mktable('/dev/sdb')

# Create a partition that spans 100% of disk space
cl.disk.mkpart('/dev/sdb', '1', '100%')

# inspect the created parition
cl.disk.list()

# Create a btrfs filesystem
cl.btrfs.create("data", "/dev/sdb1")

# make sure mount point exists
cl.system('mkdir /data')

# mount root data disk to /data
cl.disk.mount("/dev/sdb1", "/data")

#create a subvolume
cl.btrfs.subvol_create('/data/vol1')
```

## Creating a container
We will create a very basic container that only mount the root filesystem. We use this flist for testing
`https://stor.jumpscale.org/stor2/flist/ubuntu-g8os-flist/ubuntu-g8os.flist`.

```python
flist = 'https://stor.jumpscale.org/stor2/flist/ubuntu-g8os-flist/ubuntu-g8os.flist'

container_id = cl.container.create(flist, mount=mount)
container = cl.container.client(container_id)

print(container.system('ls -l /opt').get())
```
