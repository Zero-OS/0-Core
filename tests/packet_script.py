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
    ipxe_script_url = 'http://unsecure.bootstrap.gig.tech/ipxe/{}/0/development'.format(branch)
    available_facility = None
    facilities = [x.code for x in manager.list_facilities()]
    for facility in facilities:
        if manager.validate_capacity([(facility, 'baremetal_0', 1)]):
            available_facility = facility
            break

    if not available_facility:
        print('No enough resources on packet.net to create nodes')
        sys.exit(1)

    print("Available facility: %s" % available_facility)
    print('creating new machine  .. ')
    device = manager.create_device(project_id=project.id,
                                   hostname=hostname,
                                   plan='baremetal_0',
                                   operating_system='custom_ipxe',
                                   ipxe_script_url=ipxe_script_url,
                                   facility=available_facility)
    return device


def delete_device(manager):
    config = configparser.ConfigParser()
    config.read('config.ini')
    hostname = config['main']['machine_hostname']
    if hostname:
        project = manager.list_projects()[0]
        devices = manager.list_devices(project.id)
        for dev in devices:
            if dev.hostname == hostname:
                print('%s is about to be deleted' % hostname)
                for i in range(5):
                    try:
                        manager.call_api('devices/%s' % dev.id, type='DELETE')
                        print("machine has been deleted successfully")
                        break
                    except Exception as e:
                        print(e.args)
                        print(e.cause)
                        continue
                else:
                    print("%s hasn't been deleted" % hostname)


def check_status(found, branch):
    session = requests.Session()
    url = 'https://build.gig.tech/build/status'
    t1 = time.time()
    while True:
        try:
            if found:
                t2 = time.time()
                if t1+10 > t2:
                    return 'No_build_triggered'
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
    hostname = '0core{}-travis'.format(randint(100, 300))
    try:
        device = create_new_device(manager, hostname, branch=branch)
    except:
        print('device hasn\'t been created')
        raise

    print('provisioning the new machine ..')
    while True:
        dev = manager.get_device(device.id)
        time.sleep(5)
        if dev.state == 'active':
            break
    print('Giving the machine time till it finish booting')
    time.sleep(150)

    print('preparing machine for tests')
    config = configparser.ConfigParser()
    config.read('config.ini')
    config['main']['target_ip'] = dev.ip_addresses[0]['address']
    config['main']['machine_hostname'] = hostname
    with open('config.ini', 'w') as configfile:
        config.write(configfile)


if __name__ == '__main__':
    action = sys.argv[1]
    token = sys.argv[2]
    manager = packet.Manager(auth_token=token)
    print(os.system('echo $TRAVIS_EVENT_TYPE'))
    if action == 'delete':
        print('deleting the g8os machine ..')
        delete_device(manager)
    else:
        branch = sys.argv[3]
        if len(sys.argv) == 5:
            branch = sys.argv[4]
        print('branch: {}'.format(branch))
        t = check_status(True, branch)
        if t != 'No_build_triggered':
            print('build has been started at {}'.format(t))
            print('waiting for g8os build to pass ..')
            check_status(False, branch)
            time.sleep(2)
            url2 = 'https://build.gig.tech/build/history'
            session = requests.Session()
            res_hs = session.get(url2)
            if res_hs.json()[0]['started'] == t:
                if res_hs.json()[0]['status'] == 'success':
                    create_pkt_machine(manager, branch)
                else:
                    print('build has failed')
            else:
                print('build wasn\'t found in the history page')
        else:
            create_pkt_machine(manager, branch)
