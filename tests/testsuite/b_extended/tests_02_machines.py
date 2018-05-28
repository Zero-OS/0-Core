from utils.utils import BaseTest
from random import randint
import unittest
import time

class ExtendedMachines(BaseTest):

    def __init__(self, *args, **kwargs):
        super(ExtendedMachines, self).__init__(*args, **kwargs)
        self.check_g8os_connection(ExtendedMachines)
        containers = self.client.container.find('ovs')
        ovs_exist = [key for key, value in containers.items()]
        if not ovs_exist:
            ovs = self.create_container(self.ovs_flist, host_network=True, storage=self.storage, tags=['ovs'], privileged=True)
            self.ovscl = self.client.container.client(ovs)
            time.sleep(2)
            self.ovscl.json('ovs.bridge-add', {"bridge": "backplane"})
            self.ovscl.json('ovs.vlan-ensure', {'master': 'backplane', 'vlan': 2000, 'name': 'vxbackend'})
        else:
            ovs = int(ovs_exist[0])
            self.ovscl = self.client.container.client(ovs)

    def setUp(self):
        super(ExtendedMachines, self).setUp()
        self.check_g8os_connection(ExtendedMachines)

    def test001_kvm_add_remove_nics(self):
        """ g8os-035

        *Test case for testing adding and removing nics for vms*

        **Test Scenario:**

        #. Create Virtual machine (vm1)
        #. Create vlan (v1) and specific name
        #. create vxlan (vx1) with specific name
        #. Create bridge with certain name
        #. Connect the vm to all these nics types, should succeed
        #. Connect the vm to all these nics types again, should fail
        #. Deattach all these nics, should succeed
        #. Delete (vm1)
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create Virtual machine (vm1)')
        vm_name = self.rand_str()
        vm_uuid = self.create_vm(name=vm_name)

        self.lg('Create vlan (v1) and specific name')
        t1 = randint(1, 4094)
        bn1 = self.rand_str()
        self.ovscl.json('ovs.vlan-ensure', {'master': 'backplane', 'vlan': t1, 'name': bn1})

        self.lg('create vxlan (vx1) with specific name')
        vx1_id = randint(20000, 30000)
        vxbridge = self.rand_str()
        self.ovscl.json('ovs.vxlan-ensure', {'master': 'vxbackend', 'vxlan': vx1_id, 'name': vxbridge})

        self.lg('create bridge with certain name')
        bn2 = self.rand_str()
        self.client.bridge.create(bn2)

        self.lg('Connect the vm to all these nics types, should succeed')
        self.client.kvm.add_nic(vm_uuid, 'vlan', id=str(t1))
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Net']), 1)
        self.client.kvm.add_nic(vm_uuid, 'vxlan', id=str(vx1_id))
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Net']), 2)
        self.client.kvm.add_nic(vm_uuid, 'bridge', id=bn2)
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Net']), 3)

        self.lg('Connect the vm to all these nics types again, should fail')
        with self.assertRaises(RuntimeError):
            self.client.kvm.add_nic(vm_uuid, 'vlan', id=str(t1))
        with self.assertRaises(RuntimeError):
            self.client.kvm.add_nic(vm_uuid, 'vxlan', id=str(vx1_id))
        with self.assertRaises(RuntimeError):
            self.client.kvm.add_nic(vm_uuid, 'bridge', id=bn2)

        self.lg('Deattach all these nics, should succeed')
        self.client.kvm.remove_nic(vm_uuid, 'vlan', id=str(t1))
        self.client.kvm.remove_nic(vm_uuid, 'vxlan', id=str(vx1_id))
        time.sleep(2)
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Net']), 1)

        self.lg('Delete (vm1)')
        self.client.kvm.destroy(vm_uuid)

        self.lg('{} ENDED'.format(self._testID))

    def test002_kvm_attach_deattach_disks(self):
        """ g8os-036

        *Test case for testing attaching and deattaching disks for vms*

        **Test Scenario:**

        #. Create Virtual machine (vm1)
        #. Create loop device (L1)
        #. Attach L1 to vm1, should succeed
        #. Attach L1 to vm1 again, vm1 should still see L1 as one vdisk
        #. Deattach L1 from vm1, should succeed
        #. Delete (vm1)
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Destroy any vm on the system')
        self.client.bash('virsh list --all --name | xargs -n 1 virsh destroy')

        self.lg('Create Virtual machine (vm1)')
        vm_name = self.rand_str()
        vm_uuid = self.create_vm(name=vm_name)

        self.lg('Create loop device (L1)')
        loop_dev = self.setup_loop_devices(['bd0'], '500M', deattach=True)[0]

        self.lg('Attach L1 to vm1, should succeed')
        l = len(self.client.kvm.info(vm_uuid)['Block'])
        self.client.kvm.attach_disk(vm_uuid, {'url': loop_dev})
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l + 1)

        self.lg('Attach L1 to vm1 again, vm1 should still see L1 as one vdisk')
        with self.assertRaises(RuntimeError):
            self.client.kvm.attach_disk(vm_uuid, {'url': loop_dev})
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l + 1)

        self.lg('Deattach L1 from vm1, should succeed')
        time.sleep(3)
        self.client.kvm.detach_disk(vm_uuid, {'url': loop_dev})
        time.sleep(2)
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l)

        self.lg('Delete (vm1)')
        self.client.kvm.destroy(vm_uuid)

        self.lg('{} ENDED'.format(self._testID))

    def test_003_containers_backup_restore(self):
        """ g8os-037

        *Test case for container backup and restore*

        **Test Scenario:**

        #. Create container C1 using small size image
        #. Create restic repo, should succeed
        #. Backup the container using fake repo, should fail
        #. Backup the container C1 image, should succeed
        #. Make full restore to the conatiner, should succeed
        #. Check that the restored files are the same as the original backup
        #. Restore with fake snapshot id, should fail
        #. Create container with restored data only, and change the nics
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create container C1 using small size image')
        C1 = self.create_container(root_url=self.smallsize_img, storage=self.storage, privileged=True)

        self.lg('Create restic repo, should succeed')
        self.client.bash('echo rooter > /password')
        repo = 'repo' + str(randint(1, 500))
        out = self.client.bash('restic init --repo /var/cache/%s --password-file /password'% repo).get()
        self.assertEqual(out.state, 'SUCCESS')

        self.lg('Backup the container using fake repo, should fail')
        fake_repo = 'fake_repo' + str(randint(500, 1000))
        with self.assertRaises(Exception):
            self.client.container.backup(C1, 'file:///var/cache/%s?password=rooter'% fake_repo).get()

        self.lg('Backup the container C1 image, should succeed')
        url = 'file:///var/cache/%s?password=rooter' % repo
        job = self.client.container.backup(C1, url)
        snapshot = job.get(30)
        self.assertIsNotNone(snapshot)

        self.lg('Restore with fake snapshot id, should fail')

        self.lg('Make full restore to the conatiner, should succeed')
        res_url = url + '#{}'.format(snapshot)
        cid = self.client.container.restore(res_url).get()
        containers = self.client.container.list()
        new_url = 'restic:{}'.format(res_url)
        self.assertEqual(containers[str(cid)]['container']['arguments']['root'], new_url)

        self.lg('Check that the restored files are the same as the original backup')
        ccl = self.client.container.client(cid)
        self.assertEqual(ccl.filesystem.list("/bin")[0]['name'], 'nbdserver')

        self.lg('Restore with fake snapshot id, should fail')
        with self.assertRaises(Exception):
            self.client.container.restore(res_url + 'b').get()

        self.lg('Create container with restored data only, and change the nics')
        nics = [{'type': 'default'}]
        cid2 = self.create_container(new_url, nics=nics)
        ccl2 = self.client.container.client(cid2)
        nic_type = self.client.container.list()[str(cid2)]['container']['arguments']['nics'][0]['type']
        self.assertEqual(nic_type, 'default')
        self.assertEqual(ccl2.filesystem.list("/bin")[0]['name'], 'nbdserver')

        self.lg('{} ENDED'.format(self._testID))

    @unittest.skip('https://github.com/zero-os/0-core/issues/679')
    def test_004_container_passthrough_nic(self):
        """ g8os-043

        *Test case for testing nic of type passthrough of a container*

        **Test Scenario:**

        #. Create a bridge (B1).
        #. Check B1 is part of the core0 nics, should succeed.
        #. Create container (C1) and pass B1 as a nic of type passthrough to C1, should fail.
        #. Check that B1 hasn't been removed from core0 nics.
        #. Check that B1 hasn't been added to C1 nics.
        #. Create dunnmy device (D1), should succeed.
        #. Add D1 as a nic of type passthrough to C1.
        #. Check that D1 has been removed from core0 nics.
        #. Check that D1 has been added to C1 nics.
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create a bridge (B1).')
        bridge_name = self.rand_str()
        self.client.bridge.create(bridge_name)

        self.lg('Check B1 is part of the core0 nics, should succeed.')
        flag = [True for n in self.client.info.nic() if n['name'] == bridge_name]
        self.assertTrue(flag)

        self.lg('Create container (C1) and pass B1 as a nic of type passthrough to C1, should fail')
        nic = [{"type": "passthrough", "id": bridge_name, "name": bridge_name}]
        c1 = self.create_container(root_url=self.root_url, storage=self.storage, nics=nic)
        c1_client = self.client.container.client(c1)

        self.lg("Check that B1 hasn't been removed from core0 nics.")
        flag = [True for n in self.client.info.nic() if n['name'] == bridge_name]
        self.assertTrue(flag)

        self.lg("Check that B1 hasn't been added to C1 nics.")
        flag = [True for n in c1_client.info.nic() if n['name'] == bridge_name]
        self.assertFalse(flag)

        self.lg('Create dunnmy device (D1), should succeed.')
        nic_name = self.rand_str()
        self.client.bash('ip l a {} type dummy'.format(nic_name)).get()
        flag = [True for n in self.client.info.nic() if n['name'] == nic_name]
        self.assertTrue(flag)

        self.lg('Add D1 as a nic of type passthrough to C1.')
        nic = [{"type": "passthrough", "id": nic_name, "name": nic_name}]
        c1_client.container.nic_add(c1, nic)

        self.lg('Check that D1 has been removed from core0 nics.')
        flag = [True for n in self.client.info.nic() if n['name'] == nic_name]
        self.assertFalse(flag)

        self.lg('Check that D1 has been added to C1 nics.')
        flag = [True for n in c1_client.info.nic() if n['name'] == bridge_name]
        self.assertTrue(flag)

        self.lg('{} ENDED'.format(self._testID))

    def test_005_containers_macvlan_nic(self):
        """ g8os-044

        *Test case for testing nic of type macvlan*

        **Test Scenario:**

        #. Create dummy device (D1), should succeed.
        #. Create container (C1) and pass D1 as a nic of type macvlan to C1.
        #. Get the interface name (I1) that C1 is attached to as well as its mac_address.
        #. Create container (C2) and pass D1 as a nic of type macvlan to C2.
        #. Get the interface name (I2) that C2 is attached to as well as its mac_address.
        #. Assert that interfaces names are the same while the mac addresses are different.
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create dummy device (D1), should succeed.')
        phy_interface = self.rand_str()
        self.client.bash('ip l a {} type dummy'.format(phy_interface)).get()
        flag = [True for n in self.client.info.nic() if n['name'] == phy_interface]
        self.assertTrue(flag)

        self.lg('Create container (C1) and pass D1 as a nic of type macvlan to C1.')
        nic = [{"type": "macvlan", "id": phy_interface}]
        c1 = self.create_container(root_url=self.root_url, storage=self.storage, nics=nic)
        c1_client = self.client.container.client(c1)

        self.lg('Get the interface name (I1) that C1 is attached to as well as its mac_address.')
        interface_name_1 = c1_client.bash('ip a | grep @ | cut -d "@" -f2 | cut -d ":" -f1').get().stdout
        mac_address_1 = c1_client.bash("ip a | grep ether | awk '{print substr($2, 1, length($2))}'").get().stdout

        self.lg('Create container (C2) and pass D1 as a nic of type macvlan to C2.')
        nic = [{"type": "macvlan", "id": phy_interface}]
        c2 = self.create_container(root_url=self.root_url, storage=self.storage, nics=nic)
        c2_client = self.client.container.client(c2)

        self.lg('Get the interface name (I2) that C2 is attached to as well as its mac_address.')
        interface_name_2 = c2_client.bash('ip a | grep @ | cut -d "@" -f2 | cut -d ":" -f1').get().stdout
        mac_address_2 = c2_client.bash("ip a | grep ether | awk '{print substr($2, 1, length($2))}'").get().stdout

        self.lg('Assert that interfaces names are the same while the mac addresses are different.')
        self.assertEqual(interface_name_1, interface_name_2)
        self.assertNotEqual(mac_address_1, mac_address_2)
