#!/bin/python

from zeroos.core0.client import Client
import sys
import time

"""
This script expect you know the IP of the core0 and you can access it from the machine running this script.
an easy way to do it is to build the initramfs with a customr zerotier network id (https://github.com/g8os/initramfs/tree/0.10.0#customize-build)
At boot core0 will connect to the zerotier network and you can assing an IP to it.
"""

CORE0IP = "INSERT CORE0 IP HERE"
ZEROTIER = "INSERT ZEROTIER NETWORK ID HERE"

def main(init=False):
    print("[+] connect to core0")
    cl = Client(CORE0IP)

    try:
        cl.ping()
    except Exception as e:
        print("cannot connect to the core0: %s" % e)
        return 1

    print("[+] prepare data disks")
    cl.system('mkdir -p /dev/mongodb_storage').get()

    if init:
        cl.btrfs.create('mongodb_storage', ['/dev/sda'])

    disks = cl.disk.list().get('blockdevices', [])
    if len(disks) < 1:
        print("[-] need at least one data disk available")
        return

    disks_by_name = {d['name']: d for d in disks}
    if disks_by_name['sda']['mountpoint'] is None:
        print("[+] mount disk")
        cl.disk.mount('/dev/sda', '/dev/mongodb_storage', [''])

    try:
        print("[+] create container")
        container_id = cl.container.create('https://stor.jumpscale.org/stor2/flist/ubuntu-g8os-flist/mongodb-g8os.flist',
                                           mount={"/dev/mongodb_storage": "/mnt/data"},
                                           zerotier=ZEROTIER).get()
        print("[+] container created, ID: %s" % container_id)
    except Exception as e:
        print("[-] error during container creation: %s" % e)
        return 1

    container = cl.container.client(container_id)

    print("[+] get zerotier ip")
    container_ip = get_zerotier_ip(container)

    print("[+] configure mongodb")
    container.system("bash -c 'echo DAEMONUSER=\"root\" > /etc/default/mongodb'").get()
    container.system("sed -i 's/dbpath.*/dbpath=\/mnt\/data/' /etc/mongodb.conf").get()
    container.system("sed -i '/bind.*/d' /etc/mongodb.conf").get()
    container.system("bash -c 'echo nounixsocket=true >> /etc/mongodb.conf'").get()
    print("[+] starts mongod")
    res = container.system('/etc/init.d/mongodb start').get()

    print("[+] you can connect to mongodb at %s:27017" % container_ip)


def get_zerotier_ip(container):
    i = 0

    while i < 10:
        addrs = container.info.nic()
        ifaces = {a['name']: a for a in addrs}

        for iface, info in ifaces.items():
            if iface.startswith('zt'):
                cidr = info['addrs'][0]['addr']
                return cidr.split('/')[0]
        time.sleep(2)
        i += 1

    raise TimeoutError("[-] couldn't get an ip on zerotier network")

if __name__ == '__main__':
    import argparse
    parser = argparse.ArgumentParser(description='attach disks to core0')
    parser.add_argument('--init', type=bool, default=False, const=True, required=False,
                        help='creation filesystem and subvolume', nargs='?')
    args = parser.parse_args()
    # print(args.init)
    main(init=args.init)
