# Zero-OS Bootstrap Service

The Zero-OS Bootstrap Service is available on [bootstrap.gig.tech](https://bootstrap.gig.tech).

The bootstrap service provides you with the following tools to quickly and easily get you started with **always-up-to-date** builds:

- [Kernel builds](#kernel-builds)
- [Boot files](#boot-files)
- [Autobuild Server](#auto-build)

<a id="kernel-builds"></a>
## Kernel builds

On the Zero-OS Bootstrap Service home page all available kernel builds (commits) are listed.

You'll find them in two sections:
- First, under **/latest-release/**, all most recent builds per branch are listed
- And under **/kernel/** all most recent builds are listed

In both cases the naming notation `g8os-BRANCH-COMMIT.efi` is used.

E.g. in case of `g8os-1.1.0-alpha-sandbox-cleanup-core0-a2564152.efi`:
- the branch name is `1.1.0-alpha-sandbox-cleanup-core0`
- the commit, or build number is `a2564152`

<a id="boot-files"></a>
## Boot files

Next to the most recent kernel builds, the Zero-OS bootstrap service also provides you with all other files to get the kernel booted:

- [ISO file](#iso)
- [USB image](#usb)
- [iPXE script](#ipxe)

<a id="iso"></a>
### ISO

You can request an ISO file (~2MB) which contains a bootable iPXE service which will download the requested kernel.

To generate an ISO, just follow this url: `https://bootstrap.gig.tech/iso/BRANCH/ZEROTIER-NETWORK`

For example, to use the most recent build of the `1.1.0-alpha` branch, with earth's ZeroTier network: `https://bootstrap.gig.tech/iso/1.1.0-alpha/8056c2e21c000001`

Of course, you can specify a more precise branch (for debugging purposes for instance), e.g.: `https://bootstrap.gig.tech/iso/1.1.0-alpha-changeloglevel-initramfs-035dd483/8056c2e21c000001`

<a id="usb"></a>
### USB

You can also download an USB image, ready to copy to an USB stick: `https://bootstrap.gig.tech/usb/BRANCH/ZEROTIER-NETWORK`

<a id="ipxe"></a>
### iPXE Script

Like for ISO and USB, you can request a iPXE script using: `https://bootstrap.gig.tech/ipxe/BRANCH/ZEROTIER-NETWORK`

This will provide with you with the script that gets generated for, and embedded in the ISO file and the USB image.

This is useful when you want to boot a remote machine using an iPXE script, i.e. `Packet.net` or `OVH` servers.

<a id="auto-build"></a>
## Autobuild Server

Automatic kernel builds are triggered when commits happen in the following GitHub repositories:

- [zero-os/0-core](https://github.com/zero-os/0-core)
- [zero-os/0-fs](https://github.com/zero-os/0-fs)
- [zero-os/0-initramfs](https://github.com/zero-os/0-initramfs)

The build process can be monitored here: [build.gig.tech/monitor/](https://build.gig.tech/monitor/).

The result is shown on the home page of the Zero-OS bootstrap service, discussed above.

Each time a commit is pushed to GitHub, a build request is called:
- If you push to `zero-os/0-initramfs`, a complete kernel image will be rebuilt, which can take up to **1 hour**
- If you push to `zero-os/0-core` or `zero-os/0-fs`, a pre-compiled `initramfs` image (called `baseimage`) will be used, the actual build of `core0` or `0-fs` only takes **about 3 minutes**

In order to have a **3 minutes** compilation time for cores, the build process uses a pre-compiled `initramfs` image (called `baseimage`). If no base image is found, the build will be ignored.

### Base image and branches

When you push to `initramfs`, a base image will be produced automatically at the end of the build. This base image will be tagged with the branch name. E.g. if you push to `1.1.0-alpha`, the base image will be called `1.1.0-alpha`.

When you push to `core0` or `0-fs`, a base image will be looked up that matches the branch-prefix. E.g. when pushing a commit to the `1.1.0-alpha-issue-155` the build process will use the base image `1.1.0-alpha`. In theory a base image for each of the branches should exist.

So you always **NEED** to prefix your branch with the name of an existing base image. If you would push a commit to `mybranch` instead of `1.1.0-alpha-mybranch` (forgetting/omitting the prefix), the build will not occur, and an error will be raised.
