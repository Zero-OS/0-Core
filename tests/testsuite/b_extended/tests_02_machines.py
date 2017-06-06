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
            ovs = self.create_container(self.ovs_flist, host_network=True, storage=self.storage, tags=['ovs'])
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

    @unittest.skip('bug: https://github.com/g8os/core0/issues/223')
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
        self.create_vm(name=vm_name)

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
        vm_uuid = self.get_vm_uuid(vm_name)
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
        self.client.kvm.remove_nic(vm_uuid, 'bridge', id=bn2)
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Net']), 0)

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
        self.create_vm(name=vm_name)

        self.lg('Create loop device (L1)')
        loop_dev = self.setup_loop_devices(['bd0'], '500M', deattach=True)[0]

        self.lg('Attach L1 to vm1, should succeed')
        vm_uuid = self.get_vm_uuid(vm_name)
        l = len(self.client.kvm.info(vm_uuid)['Block'])
        self.client.kvm.attach_disk(vm_uuid, {'url': loop_dev})
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l+1)

        self.lg('Attach L1 to vm1 again, vm1 should still see L1 as one vdisk')
        with self.assertRaises(RuntimeError):
            self.client.kvm.attach_disk(vm_uuid, {'url': loop_dev})
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l+1)

        self.lg('Deattach L1 from vm1, should succeed')
        time.sleep(3)
        self.client.kvm.detach_disk(vm_uuid, {'url': loop_dev})
        time.sleep(2)
        self.assertEqual(len(self.client.kvm.info(vm_uuid)['Block']), l)

        self.lg('Delete (vm1)')
        self.client.kvm.destroy(vm_uuid)

        self.lg('{} ENDED'.format(self._testID))
