from zeroos.core0 import client
import unittest
import time
import uuid
import logging
import configparser
import requests
import json


class BaseTest(unittest.TestCase):
    def __init__(self, *args, **kwargs):
        config = configparser.ConfigParser()
        config.read('config.ini')
        self.target_ip = config['main']['target_ip']
        self.zt_access_token = config['main']['zt_access_token']
        self.client = client.Client(self.target_ip)
        self.session = requests.Session()
        self.session.headers['Authorization'] = 'Bearer {}'.format(self.zt_access_token)
        self.root_url = 'https://hub.gig.tech/gig-official-apps/ubuntu1604.flist'
        self.smallsize_img = 'https://hub.gig.tech/gig-official-apps/0-disk-master.flist'
        self.ovs_flist = 'https://hub.gig.tech/gig-official-apps/ovs.flist'
        self.storage = 'ardb://hub.gig.tech:16379'
        self.client.timeout = 80
        super(BaseTest, self).__init__(*args, **kwargs)

    def setUp(self):
        self._testID = self._testMethodName
        self._startTime = time.time()
        self._logger = logging.LoggerAdapter(logging.getLogger('g8os_testsuite'),
                                             {'testid': self.shortDescription() or self._testID})


    def teardown(self):
        pass

    def lg(self, msg):
        self._logger.info(msg)

    def check_g8os_connection(self, classname):
        try:
            self.client.ping()
        except Exception as e:
            self.lg("can't reach g8os remote machine")
            print("Can't reach g8os remote machine")
            self.skipTest(classname)

    def rand_str(self):
        return str(uuid.uuid4()).replace('-', '')[1:10]

    def get_process_id(self, cmdline):
        """
        Get the id of certain process
        :param cmdline: whole command to be executed
        """
        time.sleep(1)
        processes = self.client.process.list()
        for p in processes:
           if p['cmdline'] == cmdline:
               return p['pid']
        return

    def get_job_id(self, cmd, match):
        """
        Get the id of certain job
        :param cmd: command used by the client (same as the command in job.list) ex: 'core.system'
        :param match: string to match intended command. ex: 'sleep 300'
        """
        time.sleep(2)
        jobs = self.client.job.list()
        for j in jobs:
           if j['cmd']['command'] == cmd:
              if cmd == 'core.system':
                 if j['cmd']['arguments']['name'] == match:
                    return j['cmd']['id']
              if cmd == 'bash':
                 if j['cmd']['arguments']['script'] == match:
                    return j['cmd']['id']
        return

    def stdout(self, resource):
        return resource.get().stdout.replace('\n', '').lower()

    def create_zerotier_network(self, private=False):
        url = 'https://my.zerotier.com/api/network'
        data = {'config': {'ipAssignmentPools': [{'ipRangeEnd': '10.147.19.254',
                                                  'ipRangeStart': '10.147.19.1'}],
                           'private': private,
                           'routes': [{'target': '10.147.19.0/24', 'via': None}],
                           'v4AssignMode': {'zt': True}}}

        response = self.session.post(url=url, json=data)
        response.raise_for_status()
        nwid = response.json()['id']
        return nwid

    def delete_zerotier_network(self, nwid):
        url = 'https://my.zerotier.com/api/network/{}'.format(nwid)
        self.session.delete(url=url)

    def getZtNetworkID(self):
        url = 'https://my.zerotier.com/api/network'
        r = self.session.get(url)
        if r.status_code == 200:
            for item in r.json():
                if item['type'] == 'Network':
                    return item['id']
            else:
                self.lg('can\'t find network id')
                return False
        else:
            self.lg('can\'t connect to zerotier, {}:{}'.format(r.status_code, r.content))
            return False

    def getZtNetworkOnlineMembers(self, networkId):
        url = 'https://my.zerotier.com/api/network/{}'.format(networkId)
        r = self.session.get(url)
        if r.status_code == 200:
            return r.json()['onlineMemberCount']
        else:
            self.lg('can\'t connect to zerotier, {}:{}'.format(r.status_code, r.content))
            return False

    def get_g8os_zt_ip(self, networkId):
        """
        method to get the zerotier ip address of the g8os client
        """
        nws = self.client.zerotier.list()
        for nw in nws:
            if nw['nwid'] == networkId:
                address = nw['assignedAddresses'][0]
                return address[:address.find('/')]
        else:
            self.lg('can\'t find network in zerotier.list()')

    def get_contanier_zt_ip(self, client):
        """
        method to get zerotier ip address of the g8os container
        Note: to use this method, make sure that zt is defined first in the nic
        list during creating your container, so it could be attached to etho interface
        """
        nics = client.info.nic()
        for nic in nics:
            if 'zt' in nic['name']:
                address = nic['addrs'][0]['addr']
                address = address[:address.find('/')]
                return address
        else:
            self.lg('can\'t find zerotier netowrk interface')

    def deattach_all_loop_devices(self):
        self.client.bash('modprobe loop')  # to make /dev/loop* available
        self.client.bash('umount -f /dev/loop*')  # Make sure to free all loop devices first
        for i in range(8):
            self.client.bash('losetup -d /dev/loop{}'.format(i))  # deattach all devices

    def setup_loop_devices(self, files_names, file_size, files_loc='/', deattach=False):
        """
        :param files_names: list of files names to be truncated
        :param file_size: the file size (ex: 1G)
        :param files_loc: abs path for the files (ex: /)
        :param deattach: if True, deattach all loop devices
        """
        if deattach:
            self.deattach_all_loop_devices()
        loop_devs = []
        for f in files_names:
            self.client.bash('cd {}; truncate -s {} {}'.format(files_loc, file_size, f))
            output = self.client.bash('losetup -f')
            free_l_dev = self.stdout(output)
            self.client.bash('losetup {} {}{}'.format(free_l_dev, files_loc, f))
            loop_devs.append(free_l_dev)
            self.client.bash('rm -rf {}{}'.format(files_loc, f))
        return loop_devs

    def create_container(self, root_url, storage=None, nics=[], host_network=False, tags=None, privileged=False):
        container = self.client.container.create(root_url=root_url, storage=storage, host_network=host_network, nics=nics, tags=tags, privileged=privileged)
        return container.get(30)

    def get_vm_uuid(self, vm_name):
        vm_list = self.client.kvm.list()
        for vm in vm_list:
            if vm['name'] == vm_name:
                return vm['uuid']

    def create_vm(self, name, image='Ubuntu.1604.uefi.x64.qcow2', source=None):
        img_loc = '/var/cache/images'
        if source:
            img_dn_path = source
        else:
            img_dn_path = 'ftp://pub:pub1234@ftp.aydo.com/Linux/ubuntu/Ubuntu.1604.uefi.x64.qcow2'
        flag = self.client.filesystem.exists('{}'.format(img_loc))
        if flag:
            img = self.client.filesystem.exists('{}/{}'.format(img_loc, image))
            if not img:
                rs = self.client.bash('wget {} -P {}'.format(img_dn_path, img_loc))
                self.assertEqual(rs.get(100).state, 'SUCCESS')
        else:
            rs = self.client.bash('mkdir -p {}'.format(img_loc))
            self.assertEqual(rs.get().state, 'SUCCESS')
            rs = self.client.bash('wget {} -P {}'.format(img_dn_path, img_loc))
            self.assertEqual(rs.get(100).state, 'SUCCESS')
        result = self.client.kvm.create(name=name, media=[{'url': '{}/{}'.format(img_loc, image)}])
        self.assertEqual(result.state, 'SUCCESS')

    def check_nic_exist(self, name):
        nic_lst = [True for nic in self.client.info.nic() if nic['name'] == name]
        if nic_lst:
            return len(nic_lst) # should be always one
        else:
            return False
