from utils.utils import BaseTest
import time
import unittest
from random import randint


class ExtendedNetworking(BaseTest):

    def setUp(self):
        super(ExtendedNetworking, self).setUp()
        self.check_g8os_connection(ExtendedNetworking)

    def rand_mac_address(self):
        mac_addr = ["{:02X}".format(randint(0, 255)) for x in range(6)]
        return ':'.join(mac_addr)

    def test001_zerotier(self):
        """ g8os-014
        *Test case for testing zerotier functionally*

        **Test Scenario:**
        #. Get NetworkId (N1) using zerotier API
        #. G8os client join zerotier network (N1)
        #. Create 2 containers c1, c2 and make them join (N1)
        #. Get g8os and containers zerotier ip addresses
        #. Container c1 ping g8os client, should succeed
        #. Container c1 ping Container c2, should succeed
        #. Container c2 ping g8os client, should succeed
        #. Container c2 ping Container c1, should succeed
        #. G8os client ping Container c1, should succeed
        #. G8os client ping Container c2, should succeed
        #. G8os client leave zerotier network (N1), should succeed
        #. G8os client ping Container c1, should fail
        #. G8os client ping Container c2, should fail
        #. Terminate containers c1, c2
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Get NetworkId using zerotier API')
        networkId = self.getZtNetworkID()

        self.lg('Join zerotier network (N1)')
        self.client.zerotier.join(networkId)

        self.lg('Create 2 containers c1, c2 and make them join (N1) && create there clients')
        nic = [{'type': 'default'}, {'type': 'zerotier', 'id': networkId}]
        cid_1 = self.create_container(root_url=self.root_url, storage=self.storage, nics=nic)
        cid_2 = self.create_container(root_url=self.root_url, storage=self.storage, nics=nic)
        c1_client = self.client.container.client(cid_1)
        c2_client = self.client.container.client(cid_2)

        time.sleep(40)

        self.lg('Get g8os and containers zerotier ip addresses')
        g8_ip = self.get_g8os_zt_ip(networkId)
        c1_ip = self.get_contanier_zt_ip(c1_client)
        c2_ip = self.get_contanier_zt_ip(c2_client)

        self.lg('set client time to 100 sec')
        self.client.timeout = 100

        self.lg('Container c1 ping g8os client (ip : {}), should succeed'.format(g8_ip))
        r = c1_client.bash('ping -w10 {}'.format(g8_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('Container c1 ping Container c2 (ip : {}), should succeed'.format(c2_ip))
        r = c1_client.bash('ping -w10 {}'.format(c2_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('Container c2 ping g8os client (ip : {}), should succeed'.format(g8_ip))
        r = c2_client.bash('ping -w10 {}'.format(g8_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('Container c2 ping Container c1 (ip : {}), should succeed'.format(c1_ip))
        r = c2_client.bash('ping -w10 {}'.format(c1_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('G8os client ping Container c1 (ip : {}), should succeed'.format(c1_ip))
        r = self.client.bash('ping -w10 {}'.format(c1_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('G8os client ping Container c2 (ip : {}), should succeed'.format(c2_ip))
        r = self.client.bash('ping -w10 {}'.format(c2_ip)).get()
        self.assertEqual(r.state, 'SUCCESS', r.stdout)

        self.lg('G8os client leave zerotier network (N1), should succeed')
        self.client.zerotier.leave(networkId)
        time.sleep(5)

        self.lg('G8os client ping Container c1 (ip : {}), should fail'.format(c1_ip))
        r = self.client.bash('ping -w10 {}'.format(c1_ip)).get()
        self.assertEqual(r.state, 'ERROR', r.stdout)

        self.lg('G8os client ping Container c2 (ip : {}), should fail'.format(c2_ip))
        r = self.client.bash('ping -w10 {}'.format(c2_ip)).get()
        self.assertEqual(r.state, 'ERROR', r.stdout)

        self.lg('Terminate c1, c2')
        self.client.container.terminate(cid_1)
        self.client.container.terminate(cid_2)

        self.lg('{} ENDED'.format(self._testID))

    def test002_create_bridges_with_specs_hwaddr(self):

        """ g8os-023
        *Test case for testing creating, listing, deleting bridges*

        **Test Scenario:**
        #. Create bridge (B1) with specifice hardware address (HA), should succeed
        #. List bridges, (B1) should be listed
        #. Check the created bridge hardware address equal to (HA), should succeed
        #. Delete bridge (B1), should succeed
        #. Create bridge (B2) with invalid hardware address, should fail

        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create bridge (B1) with specifice hardware address (HA), should succeed')
        bridge_name = self.rand_str()
        hardwareaddr = '32:64:7d:0b:c7:aa' # private mac
        self.client.bridge.create(bridge_name, hwaddr=hardwareaddr)

        self.lg('List bridges, (B1) should be listed')
        bridges = self.client.bridge.list()
        self.assertIn(bridge_name, bridges)

        self.lg('Check the created bridge hardware address equal to (HA), should succeed')
        nics = self.client.info.nic()
        nic = [x for x in nics if x['name'] == bridge_name]
        self.assertNotEqual(nic, [])
        self.assertEqual(nic[0]['hardwareaddr'].lower(), hardwareaddr.lower())

        self.lg('Delete bridge (B1), should succeed')
        self.client.bridge.delete(bridge_name)
        bridges = self.client.bridge.list()
        self.assertNotIn(bridge_name, bridges)

        self.lg('Create bridge (B2) with invalid hardware address, should fail')
        bridge_name = self.rand_str()
        hardwareaddr = self.rand_str()
        with self.assertRaises(RuntimeError):
            self.client.bridge.create(bridge_name, hwaddr=hardwareaddr)

        self.lg('{} ENDED'.format(self._testID))

    def test003_create_bridges_with_specs_network(self):
        """ g8os-024
        *Test case for testing creating, listing, deleting bridges*

        **Test Scenario:**
        #. Create bridge (B1) with static network and cidr (C1), should succeed
        #. Check the created bridge addresses contains cidr (C1), should succeed
        #. Create another bridge with static network and cidr (C1), should fail
        #. Delete bridge (B1), should succeed
        #. Create bridge with invalid cidr, should fail
        #. Create bridge (B2) with dnsmasq network and cidr (C2), should succeed
        #. Check the bridge (B2) addresses contains cidr (C2), should succeed
        #. Delete bridge (B2), should succeed

        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create bridge (B1) with static network and cidr (C1), should succeed')
        bridge_name = self.rand_str()
        cidr = "10.20.30.1/24"
        settings = {"cidr":cidr}
        self.client.bridge.create(bridge_name, network='static', settings=settings)

        self.lg('Check the created bridge addresses contains cidr (C1), should succeed')
        nics = self.client.info.nic()
        nic = [x for x in nics if x['name'] == bridge_name]
        self.assertNotEqual(nic, [])
        addrs = [x['addr'] for x in nic[0]['addrs']]
        self.assertIn(cidr, addrs)

        self.lg('Create another bridge with static network and cidr (C1), should fail')
        with self.assertRaises(RuntimeError):
            self.client.bridge.create(self.rand_str(), network='static', settings=settings)

        self.lg('Delete bridge (B1), should succeed')
        self.client.bridge.delete(bridge_name)
        bridges = self.client.bridge.list()
        self.assertNotIn(bridge_name, bridges)

        self.lg('Create bridge with invalid cidr, should fail')
        bridge_name = self.rand_str()
        cidr = "10.20.30.1"
        settings = {"cidr":cidr}
        with self.assertRaises(RuntimeError):
            self.client.bridge.create(bridge_name, network='static', settings=settings)

        self.lg('Create bridge (B2) with dnsmasq network and cidr (C2), should succeed')
        bridge_name = self.rand_str()
        cidr = "10.20.30.1/24"
        start = "10.20.30.2"
        end = "10.20.30.3"
        settings = {"cidr":cidr, "start":start, "end":end}
        self.client.bridge.create(bridge_name, network='dnsmasq', settings=settings)

        self.lg('Check the bridge (B2) addresses contains cidr (C2), should succeed')
        nics = self.client.info.nic()
        nic = [x for x in nics if x['name'] == bridge_name]
        self.assertNotEqual(nic, [])
        addrs = [x['addr'] for x in nic[0]['addrs']]
        self.assertIn(cidr, addrs)

        self.lg('Delete bridge (B2), should succeed')
        self.client.bridge.delete(bridge_name)
        bridges = self.client.bridge.list()
        self.assertNotIn(bridge_name, bridges)

        self.lg('{} ENDED'.format(self._testID))

    def test004_attach_bridge_to_container(self):
        """ g8os-027
        *Test case for testing creating, listing, deleting bridges*

        **Test Scenario:**
        #. Create bridge (B1) with dnsmasq network and cidr (CIDR1), should succeed
        #. Create 2 containers C1, C2 with bridge (B1), should succeed
        #. Check if each container (C1), (C2) got an ip address, should succeed
        #. Check if each container can reach the other one, should succeed
        #. Delete bridge (B1), should succeed
        #. Check if host can reach remote server (google.com) using dns
        #. Create another bridge (B2) with same cidr, should succeed
        #. Create container (C3) and check if it got ip address, should succeed
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create bridge (B1) with dnsmasq network and cidr (CIDR1), should succeed')
        bridge_name = self.rand_str()
        cidr = "20.20.30.1/24"
        ip_range = ["20.20.30.2", "20.20.30.3"]
        start = ip_range[0]
        end = ip_range[1]
        settings = {"cidr":cidr, "start":start, "end":end}
        self.client.bridge.create(bridge_name, network='dnsmasq', settings=settings)

        self.lg('Create 2 containers C1, C2 with bridge (B1), should succeed')
        nic1 = [{'type': 'bridge', 'id': bridge_name, 'config': {'dhcp': True}}]
        cid_1 = self.create_container(self.root_url, storage=self.storage, nics=nic1)
        cid_2 = self.create_container(self.root_url, storage=self.storage, nics=nic1)
        client_c1 = self.client.container.client(cid_1)
        client_c2 = self.client.container.client(cid_2)

        for container_client in [client_c1, client_c2]:
            self.lg('Check if each container (C1), (C2) got an ip address, should succeed')
            time.sleep(20)
            nics = container_client.info.nic()
            nic = [x for x in nics if x['name'] == 'eth0']
            self.assertNotEqual(nic, [])
            current_container_addr = [x['addr'] for x in nic[0]['addrs'] if x['addr'][:x['addr'].find('/')] in ip_range][0]
            self.assertNotEqual(current_container_addr, [])
            other_container_addr = [x for x in ip_range if x != current_container_addr][0]

            self.lg('Check if each container can reach the other one, should succeed')
            response = container_client.bash('ping -w10 {}'.format(other_container_addr)).get()
            self.assertEqual(response.state, 'SUCCESS', response.stderr)

        self.lg('Delete bridge (B1), should succeed')
        self.client.bridge.delete(bridge_name)
        bridges = self.client.bridge.list()
        self.assertNotIn(bridge_name, bridges)

        self.lg('Check if host can reach remote server (google.com) using dns')
        response = self.client.bash('ping -w10 google.com').get()
        self.assertEqual(response.state, 'SUCCESS', response.stderr)

        self.lg('Create another bridge (B2) with same cidr, should succeed')
        b2_name = self.rand_str()
        self.client.bridge.create(b2_name, network='dnsmasq', settings=settings)

        self.lg('Create container (C3) and check if it got ip address, should succeed')
        nic2 = [{'type': 'bridge', 'id': b2_name, 'config': {'dhcp': True}}]
        cid_3 = self.create_container(self.root_url, storage=self.storage, nics=nic2)
        client_c3 = self.client.container.client(cid_3)
        time.sleep(20)
        nics = client_c3.info.nic()
        nic = [x for x in nics if x['name'] == 'eth0']
        self.assertNotEqual(nic, [])
        current_container_addr = [x['addr'] for x in nic[0]['addrs'] if x['addr'][:x['addr'].find('/')] in ip_range][0]
        self.assertNotEqual(current_container_addr, [])

        self.lg('{} ENDED'.format(self._testID))

    def test005_create_bridges_with_specs_nat(self):
        """ g8os-028
        *Test case for testing creating, listing, deleting bridges*

        **Test Scenario:**
        #. Create bridge (B1) with nat = False/True, should succeed
        #. Create new container and attach bridge (B1) to it, should succeed
        #. Add ip to eth0 and set it up, should succeed
        #. set network interface eth0 as default route, should succeed
        #. Try to ping 8.8.8.8 when NAT is enabled, should succeed
        #. Try to ping 8.8.8.8 when NAT is disabled, should fail
        #. Delete bridge (B2), should succeed
        #. Delete container, should succeed

        """
        self.lg('{} STARTED'.format(self._testID))

        bridge_name = self.rand_str()[0:3]
        settings = {'cidr': '10.1.0.1/24'}

        for nat in [True, False]:

            self.lg('Create bridge (B1) with nat = {}, should succeed'.format(nat))
            self.client.bridge.create(bridge_name, network='static', nat=nat, settings=settings)
            time.sleep(2)

            self.lg('Create new container and attach bridge (B1) to it, should succeed')
            nic = [{'type': 'bridge', 'id': bridge_name}]
            cid = self.create_container(self.root_url, storage=self.storage, nics=nic)
            container_client = self.client.container.client(cid)
            time.sleep(2)

            self.lg('Add ip to eth0 and set it up')
            response = container_client.system('ip a a 10.1.0.2/24 dev eth0').get()
            self.assertEqual(response.state, 'SUCCESS', response.stderr)
            response = container_client.bash('ip l s eth0 up').get()
            self.assertEqual(response.state, 'SUCCESS', response.stderr)
            time.sleep(1)

            self.lg('set network interface eth0 as default route, should succeed')
            response = container_client.bash('ip route add default dev eth0 via 10.1.0.1').get()
            self.assertEqual(response.state, 'SUCCESS', response.stderr)
            time.sleep(2)

            if nat:
                self.lg('Try to ping 8.8.8.8 when NAT is enabled, should succeed')
                response = container_client.bash('ping -w3 8.8.8.8', response.stdout).get()
                self.assertEqual(response.state, 'SUCCESS', response.stdout)
            else:
                time.sleep(25)
                self.lg('Try to ping 8.8.8.8 when NAT is disabled, should fail')
                response = container_client.bash('ping -w3 8.8.8.8', response.stdout).get()
                self.assertEqual(response.state, 'ERROR', response.stdout)

            self.lg('Delete bridge (B1), should succeed')
            self.client.bridge.delete(bridge_name)
            bridges = self.client.bridge.list()
            self.assertNotIn(bridge_name, bridges)

            self.lg('Delete container, should succeed')
            self.client.container.terminate(cid)

        self.lg('{} ENDED'.format(self._testID))
