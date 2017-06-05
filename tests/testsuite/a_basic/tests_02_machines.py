from utils.utils import BaseTest
import time
import unittest


class Machinetests(BaseTest):

    def setUp(self):
        super(Machinetests, self).setUp()
        self.check_g8os_connection(Machinetests)

    def test001_create_destroy_list_kvm(self):
        """ g8os-009

        *Test case for testing creating, listing and destroying VMs*

        **Test Scenario:**

        #. Check that system support hardware virtualization
        #. Create virtual machine (VM1), should succeed
        #. Create another vm with the same name, should fail
        #. List all virtual machines and check that VM1 is there
        #. Create another virtual machine with the same kvm domain, should fail
        #. Destroy VM1, should succeed
        #. List the virtual machines, VM1 should be gone
        #. Destroy VM1 again, should fail
        """

        self.lg('{} STARTED'.format(self._testID))
        vm_name = self.rand_str()

        self.lg('- Check that it support hardware virtualization ')
        responce = self.client.info.cpu()
        vmx = ['vmx'or'svm' in dec['flags'] for dec in responce]
        self.assertGreater(len(vmx), 0)

        self.lg('- Create virtual machine {} , should succeed'.format(vm_name))
        self.create_vm(name=vm_name)

        self.lg('Create another vm with the same name, should fail')
        with self.assertRaises(RuntimeError):
            self.create_vm(name=vm_name)

        self.lg('- List all virtual machines and check that VM {} is there '.format(vm_name))
        Vms_list = self.client.kvm.list()
        self.assertTrue(any(vm['name'] == vm_name for vm in Vms_list))

        self.lg('- create another virtual machine with the same kvm domain ,should fail')
        with self.assertRaises(RuntimeError):
            self.create_vm(name=vm_name)

        self.lg('- Destroy VM {}'.format(vm_name))
        vm_uuid = self.get_vm_uuid(vm_name)
        self.client.kvm.destroy(vm_uuid)

        self.lg('- List the virtual machines , VM {} should be gone'.format(vm_name))
        Vms_list = self.client.kvm.list()
        self.assertFalse(any(vm['name'] == vm_name for vm in Vms_list))

        self.lg('- Destroy VM {} again should fail'.format(vm_name))
        with self.assertRaises(RuntimeError):
            self.client.kvm.destroy(vm_uuid)

        self.lg('{} ENDED'.format(self._testID))

    def test002_create_list_delete_containers(self):
        """ g8os-010
        *Test case for testing creating, listing and deleting containers*

        **Test Scenario:**
        #. Create a new container (C1), should succeed
        #. List all containers and check that C1 is there
        #. Destroy C1, should succeed
        #. List the containers, C1 should be gone
        #. Destroy C1 again, should fail

        """
        self.lg('{} STARTED'.format(self._testID))
        self.lg('Create a new container (C1)')
        C1 = self.create_container(root_url=self.root_url, storage=self.storage)

        self.lg('List all containers and check that C1 {}is there'.format(C1))
        containers = self.client.container.list()
        self.assertTrue(str(C1) in containers)

        self.lg('Destroy C1 {}, should succeed'.format(C1))
        res = self.client.container.terminate(C1)
        self.assertEqual(res, None)

        self.lg('List the containers, C1 {} should be gone'.format(C1))
        time.sleep(0.5)
        containers = self.client.container.list()
        self.assertFalse(str(C1) in containers)

        self.lg('Destroy C1 again, should fail')
        with self.assertRaises(RuntimeError):
            self.client.container.terminate(C1)

        self.lg('{} ENDED'.format(self._testID))

    def test003_deal_with_container_client(self):
        """ g8os-011

        *Test case for testing dealing with container client*

        **Test Scenario:**

        #. Create a new container (C1), should succeed
        #. Get container(C1) client
        #. Use container client  to create  folder using system, should succeed
        #. Use container client to check folder is exist using bash
        #. Use G8os client to check the folder is created only in container
        #. Use container client to delete created folder
        #. Destroy C1, should succeed

        """
        self.lg('{} STARTED'.format(self._testID))
        self.lg('Create a new container (C1), and make sure its exist')
        C1 = self.create_container(root_url=self.root_url, storage=self.storage)
        containers = self.client.container.list()
        self.assertTrue(str(C1) in containers)

        self.lg('Get container client(C1)')
        C1_client = self.client.container.client(C1)

        self.lg('Use container client to create  folder using system, should succeed')
        folder = self.rand_str()
        C1_client.system('mkdir {}'.format(folder))
        time.sleep(1.5)

        self.lg('Use container client to check folder is exist using bash')
        output = C1_client.bash('ls | grep {}'.format(folder))
        self.assertEqual(self.stdout(output), folder)

        self.lg('Check that the folder is created only in container')
        output2 = self.client.bash('ls | grep {}'.format(folder))
        self.assertEqual(self.stdout(output2), '')

        self.lg('Remove the created folder using bash,check that it removed ')
        C1_client.bash('rm -rf {}'.format(folder))
        time.sleep(1)
        output = self.client.bash('ls | grep {}'.format(folder))
        self.assertEqual(self.stdout(output), '')

        self.lg('Destroy C1, should succeed')
        self.client.container.terminate(C1)

        self.lg('{} ENDED'.format(self._testID))
