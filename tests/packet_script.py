#!/usr/bin/python3
from random import randint
import packet
import time
import os
from zeroos.core0 import client
import configparser
import sys
import requests


def create_new_device(manager, hostname='zero2s', branch='master'):
    project = manager.list_projects()[0]
    ipxe_script_url = 'https://bootstrap.gig.tech/ipxe/{}/abcdef/console=ttyS1,115200n8%20debug'.format(branch)
    print('creating new machine  .. ')
    device = manager.create_device(project_id=project.id,
                                   hostname=hostname,
                                   plan='baremetal_0',
                                   operating_system='custom_ipxe',
                                   ipxe_script_url=ipxe_script_url,
                                   facility='ams1')
    return device


def delete_device(manager, hostname='zero2s'):
    project = manager.list_projects()[0]
    devices = manager.list_devices(project.id)
    for dev in devices:
        if dev.hostname == hostname:
            device_id = dev.id
    params = {
             "hostname": hostname,
             "description": "string",
             "billing_cycle": "hourly",
             "userdata": "",
             "locked": False,
             "tags": []
             }
    if device_id:
        manager.call_api('devices/%s' % device_id, type='DELETE', params=params)


def mount_disks(config):
    target_ip = config['main']['target_ip']
    time.sleep(10)
    cl = client.Client(target_ip)
    cl.timeout = 100
    cl.btrfs.create('storage', ['/dev/sda'])
    cl.disk.mount('/dev/sda', '/var/cache', options=[""])


def check_status(found, branch):
    session = requests.Session()
    url = 'https://build.gig.tech/build/status'
    while True:
        try:
            res_st = session.get(url)
            t = res_st.json()['zero-os/0-core/{}'.format(branch)]['started']
            if found:
                return t
        except:
            if found:
                continue
            break
    time.sleep(1)


def create_pkt_machine(manager, branch):
    #hostname = 'g8os{}'.format(randint(100, 300))
    try:
        device = create_new_device(manager, branch=branch)
    except:
        print('device hasn\'t been created')
        raise

    print('provisioning the new machine ..')
    while True:
        dev = manager.get_device(device.id)
        if dev.state == 'active':
            break
    time.sleep(5)

    print('preparing machine for tests')
    config = configparser.ConfigParser()
    config.read('config.ini')
    config['main']['target_ip'] = dev.ip_addresses[0]['address']
    with open('config.ini', 'w') as configfile:
        config.write(configfile)
    mount_disks(config)


if __name__ == '__main__':
    action = sys.argv[1]
    token = sys.argv[2]
    manager = packet.Manager(auth_token=token)
    if action == 'delete':
        print('deleting the g8os machine ..')
        delete_device(manager)
    else:
        branch = sys.argv[3]
        if len(sys.argv) == 5:
            branch = sys.argv[4]
        print('branch: {}'.format(branch))
        url2 = 'https://build.gig.tech/build/history'
        session = requests.Session()
        t = check_status(True, branch)
        print('build has been started at {}'.format(t))
        print('waiting for g8os build to pass ..')
        check_status(False, branch)
        time.sleep(2)
        res_hs = session.get(url2)
        if res_hs.json()[0]['started'] == t:
            if res_hs.json()[0]['status'] == 'success':
                create_pkt_machine(manager, branch)
            else:
                print('build has failed')
        else:
            print('build wasn\'t found in the history page')
