# Booting from USB

On a EFI enabled machine, it's really easy to boot the Zero-OS kernel. No boot loader needed.

All you need is to copy the Zero-OS boot image (kernel) on the `FAT32` partition of the boot device, for instance a USB device.

## Creating an USB device with the Zero-OS boot image on Linux

We assume that your USB device is `/dev/sdc`, you can make boot the kernel as following:

- Create a FAT32 partition (this will erase the whole device): `mkfs.vfat /dev/sdc`
- Mount the partition: `mount /dev/sdc /mnt/g8os-usb`
- Create the EFI directories: `mkdir -p /mnt/g8os-usb/EFI/BOOT`
- Copy the kernel: `cp v /mnt/g8os-usb/EFI/BOOT/BOOTX64.EFI`
- Unmount the USB device: `umount /mnt/g8os-usb`

You can now boot from the USB device on your EFI enabled system.
