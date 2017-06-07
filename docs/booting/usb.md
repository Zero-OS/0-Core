# Booting from USB

Two options:
- [Boot without a boot loader](#no-bootloader)
- [Boot with a boot loader (iPXE)](#ipxe)

<a id="no-bootloader"></a>
## Booting without a boot loader

On an EFI enabled machine, it's really easy to boot the Zero-OS kernel.

All you need to do is to copy a Zero-OS kernel on the `FAT32` partition of the boot device, for instance an USB device.

So you first step will be to get the kernel, options are:
- Create your own image, as documented in [Building your own Zero-OS Boot Image](../building/building.md)
- Get a kernel build from the [Zero-OS Bootstrap Service](https://bootstrap.gig.tech)

The second option is your quickest option:
```bash
curl -o zero-os-master.efi https://bootstrap.gig.tech/kernel/zero-os-master.efi
```

For downloading other versions of the kernel from the Zero-OS Bootstrap Service, see the [Zero-OS Bootstrap Service](bootstrap/bootstrap.md) documentation.

Next you'll need to go through follow steps:
- [Create a FAT32 partition](#fat32)
- [Mount the partition](#mount)
- [Create the EFI directories](#efi-dirs)
- [Copy the kernel](#copy)
- [Unmount the USB device](#unmount)

<a id="fat32"></a>
Assuming that your USB device is `/dev/sdc`, you first need to create a FAT32 partition (this will erase the whole device):
``` bash
mkfs.vfat /dev/sdc
```

<a id="mount"></a>
Then mont the partition:
```bash
mount /dev/sdc /mnt/g8os-usb
```

<a id="efi-dirs"></a>
Create the EFI directories:
```bash
mkdir -p /mnt/g8os-usb/EFI/BOOT
```

<a id="copy"></a>
Copy the kernel:
```bash
cp zero-os-master.efi /mnt/g8os-usb/EFI/BOOT/BOOTX64.EFI
```

<a id="unmount"></a>
Unmount the USB device:
```bash
umount /mnt/g8os-usb
```

You can now boot from the USB device on your EFI enabled system.


<a id="ipxe"></a>
# Use a boot loader

Using a boot loader has several advantages, including:
- You can download the kernel at boot time
- You can pass kernel argument at at boot time, e.g. passing the ZeroTier network ID.


As documented in [Zero-OS Bootstrap Service](../bootstrap/bootstrap.md) downloading the bootable USB image is simple:

```
BRANCH="zero-os-master"
ZEROTIER_NETWORK="<your_zerotier_network>"
curl -o zero-os.iso https://bootstrap.gig.tech/usb/$BRANCH/$ZEROTIER_NETWORK
```

This image includes iPXE and following iPXE script:
```
#!ipxe
dhcp
chain https://bootstrap.gig.tech/kernel/zero-os-master.efi zerotier=$ZEROTIER_NETWORK
```
