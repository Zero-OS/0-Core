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
        networkId = self.getZtNetworkID()

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

        self.lg('{} ENDED'.format(self._testID))

    @unittest.skip('bug: https://github.com/zero-os/0-core/issues/291')
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
