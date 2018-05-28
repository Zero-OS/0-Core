from utils.utils import BaseTest
import time
import unittest


class BasicNetworking(BaseTest):

    def setUp(self):
        super(BasicNetworking, self).setUp()
        self.check_g8os_connection(BasicNetworking)

    def test001_join_leave_list_zerotier(self):
        """ g8os-012
        *Test case for testing joining, listing, leaving zerotier networks*

        **Test Scenario:**
        #. Get NetworkId using zerotier API
        #. Join zerotier network (N1), should succeed
        #. List zerotier network
        #. Join fake zerotier network (N1), should fail
        #. Leave zerotier network (N1), should succeed
        #. List zerotier networks, N1 should be gone
        #. Leave zerotier network (N1), should fail
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Get NetworkId using zerotier API')
        networkId = self.create_zerotier_network()

        try:
            self.lg('Join zerotier network (N1), should succeed')
            self.client.zerotier.join(networkId)

            self.lg('List zerotier network')
            r = self.client.zerotier.list()
            self.assertIn(networkId, [x['nwid'] for x in r])

            self.lg('Join fake zerotier network (N1), should fail')
            with self.assertRaises(RuntimeError):
                self.client.zerotier.join(self.rand_str())

            self.lg('Leave zerotier network (N1), should succeed')
            self.client.zerotier.leave(networkId)

            self.lg('List zerotier networks, N1 should be gone')
            r = self.client.zerotier.list()
            self.assertNotIn(networkId, [x['nwid'] for x in r])

            self.lg('Leave zerotier network (N1), should fail')
            with self.assertRaises(RuntimeError):
                self.client.zerotier.leave(networkId)
        finally:
            self.delete_zerotier_network(networkId)

        self.lg('{} ENDED'.format(self._testID))

    def test002_create_delete_list_bridges(self):
        """ g8os-013
        *Test case for testing creating, listing, deleting bridges*

        **Test Scenario:**
        #. Create bridge (B1), should succeed
        #. List bridges, B1 should be listed
        #. Create bridge with same name of (B1), should fail
        #. Delete bridge B1, should succeed
        #. List bridges, B1 should be gone
        #. Delete bridge B1, should fail
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create bridge (B1), should succeed')
        bridge_name = self.rand_str()
        self.client.bridge.create(bridge_name)

        self.lg('List bridges, B1 should be listed')
        response = self.client.bridge.list()
        self.assertIn(bridge_name, response)

        self.lg('Create bridge with same name of (B1), should fail')
        with self.assertRaises(RuntimeError):
            self.client.bridge.create(bridge_name)

        self.lg('Delete bridge B1, should succeed')
        self.client.bridge.delete(bridge_name)

        self.lg('List bridges, B1 should be gone')
        response = self.client.bridge.list()
        self.assertNotIn(bridge_name, response)

        self.lg('Delete bridge B1, should fail')
        with self.assertRaises(RuntimeError):
            self.client.bridge.delete(bridge_name)

        self.lg('{} ENDED'.format(self._testID))

    def test003_add_remove_list_nics_for_bridge(self):
        """ g8os-045
        *Test case for adding, removing and listing nics for a bridges*

        **Test Scenario:**
        #. Create bridge (B1), should succeed.
        #. List B1 nics, should be empty.
        #. Create an nic (N1) for core0, should succeed.
        #. Add nic (N1) to bridge (B1), should succeed.
        #. List B1 nics, N1 should be found.
        #. Remove N1, should succed.
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create bridge (B1), should succeed')
        bridge_name = self.rand_str()
        self.client.bridge.create(bridge_name)

        self.lg('List B1 nics, should be empty.')
        nics = self.client.bridge.nic_list(bridge_name)
        self.assertFalse(nics)

        self.lg('Create an nic (N1) for core0, should succeed.')
        nic_name = self.rand_str()
        self.client.bash('ip l a {} type dummy'.format(nic_name)).get()
        nic = [n for n in self.client.info.nic() if n['name'] == nic_name]
        self.assertTrue(flag)

        self.lg('Add nic (N1) to bridge (B1), should succeed.')
        self.client.bridge.nic_add(bridge_name, nic_name)

        self.lg('List B1 nics, N1 should be found.')
        nics = self.client.bridge.nic_list(bridge_name)
        self.assertEqual(len(nics), 1)
        self.assertEqual(nics[0], nic_name)

        self.lg('Remove N1, should succed.')
        self.client.bridge.nic_remove(nic_name)
        nics = self.client.bridge.nic_list(bridge_name)
        self.assertFalse(nics)

        self.lg('{} ENDED'.format(self._testID))
