#!/bin/python

from zeroos.core0.client import Client
import sys
import time

"""
This script expect you know the IP of the core0 and you can access it from the machine running this script.
an easy way to do it is to build the initramfs with a customr zerotier network id (https://github.com/g8os/initramfs/tree/0.10.0#customize-build)
At boot core0 will connect to the zerotier network and you can assing an IP to it.
"""

SSHKEY = "INSERT YOUR PUBLIC SSH KEY HERE"
CORE0IP = "INSERT CORE0 IP HERE"
ZEROTIER = "INSERT NETWORK ID HERE"

def main():
    print("[+] connect to core0")
    cl = Client(CORE0IP)

    try:
        cl.ping()
    except Exception as e:
        print("cannot connect to the core0: %s" % e)
        return 1

    try:
        print("[+] create container")
        container_id = cl.container.create('https://stor.jumpscale.org/stor2/flist/ubuntu-g8os-flist/ubuntu-g8os.flist', zerotier=ZEROTIER).get()
        print("[+] container created, ID: %s" % container_id)
    except Exception as e:
        print("[-] error during container creation: %s" % e)
        return 1

    container = cl.container.client(container_id)

    print("[+] authorize ssh key")
    container.system('bash -c "echo \'%s\' > /root/.ssh/authorized_keys"' % SSHKEY)
    print("[+] start ssh daemon")
    container.system('/etc/init.d/ssh start')

    print("[+] get zerotier ip")
    container_ip = get_zerotier_ip(container)

    print("[+] you can ssh into your container at root@%s" % container_ip)


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
    main()
