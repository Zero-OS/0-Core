#!/usr/bin/python3
from random import randint
import packet
import time
import os
from zeroos.core0 import client
import configparser
import sys
import requests


def create_new_device(manager, hostname, branch='master'):
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


def delete_device(manager, hostname, device_id):
    params = {
             "hostname": hostname,
             "description": "string",
             "billing_cycle": "hourly",
             "userdata": "",
             "locked": False,
             "tags": []
             }
    manager.call_api('devices/%s' % device_id, type='DELETE', params=params)


def mount_disks(config):
    target_ip = config['main']['target_ip']
    cl = client.Client(target_ip)
    cl.timeout = 100
    cl.btrfs.create('storage', ['/dev/sda'])
    cl.disk.mount('/dev/sda', '/var/cache', options=[""])


def check_status(found):
    session = requests.Session()
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


def run_tests(branch):
    token = sys.argv[2]
    manager = packet.Manager(auth_token=token)
    hostname = 'g8os{}'.format(randint(100, 300))
    try:
        device = create_new_device(manager, hostname, branch=branch)
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

    print('start running g8os tests .. ')
    os.system('nosetests -v -s testsuite')

    print('deleting the g8os machine ..')
    delete_device(manager, hostname, device.id)


if __name__ == '__main__':
    branch = sys.argv[1]
    pull_req_branch = sys.argv[3]
    if pull_req_branch:
        branch = pull_req_branch
    print('branch: {}'.format(branch))
    url = 'https://build.gig.tech/build/status'
    url2 = 'https://build.gig.tech/build/history'
    session = requests.Session()
    t = check_status(True)
    print('build has been started at {}'.format(t))
    print('waiting for g8os build to pass ..')
    check_status(False)
    time.sleep(2)
    res_hs = session.get(url2)
    if res_hs.json()[0]['started'] == t:
        if res_hs.json()[0]['status'] == 'success':
            run_tests(branch)
        else:
            print('build has failed')
    else:
        print('build wasn\'t found in the history page')
