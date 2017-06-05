from utils.utils import BaseTest
import unittest


class DisksTests(BaseTest):

    def setUp(self):
        super(DisksTests, self).setUp()
        self.check_g8os_connection(DisksTests)

    def create_btrfs(self, second_btrfs=False):
        self.lg('Create Btrfs file system (Bfs1), should succeed')
        self.label = self.rand_str()
        self.loop_dev_list = self.setup_loop_devices(['bd0', 'bd1'], '500M', deattach=True)
        self.lg('Mount the btrfs filesystem (Bfs1)')
        self.client.btrfs.create(self.label, self.loop_dev_list)
        self.mount_point = '/mnt/{}'.format(self.rand_str())
        self.client.bash('mkdir -p {}'.format(self.mount_point))
        self.client.disk.mount(self.loop_dev_list[0], self.mount_point, [""])
        if second_btrfs:
            self.label2 = self.rand_str()
            self.loop_dev_list2 = self.setup_loop_devices(['bd2', 'bd3'], '500M', deattach=False)
            self.lg('Mount the btrfs filesystem (Bfs2)')
            self.client.btrfs.create(self.label2, self.loop_dev_list2)
            self.mount_point2 = '/mnt/{}'.format(self.rand_str())
            self.client.bash('mkdir -p {}'.format(self.mount_point2))
            self.client.disk.mount(self.loop_dev_list2[0], self.mount_point2, [""])

    def destroy_btrfs(self):
        self.lg('Remove all loop devices')
        self.deattach_all_loop_devices()

    def bash_disk_info(self, keys, diskname):
        diskinf = {}
        upper_values = ['disc-gran', 'disc-max', 'wsame', 'serial']
        info = self.client.bash('lsblk -d dev/{} -O -P '.format(diskname))
        info = self.stdout(info)
        lines = info.split()
        for key in keys:
            for line in lines:
                if key == line[:line.find('=')]:
                    value = line[line.find('=')+2:-1]
                    if value == '':
                        value = None
                    if key in upper_values:
                        value = value.upper()
                    diskinf[key] = value
                    break

        diskinf['model'] = self.stdout(self.client.bash('cat /sys/block/{}/device/model'.format(diskname))).upper()
        diskinf['vendor'] = self.stdout(self.client.bash('cat /sys/block/{}/device/vendor'.format(diskname))).upper()

        logical_block_size = int(self.stdout(self.client.bash('cat /sys/block/{}/queue/logical_block_size '.format(diskname))))
        size = int(self.stdout(self.client.bash('cat /sys/block/{}/size '.format(diskname))))
        diskinf['size'] = logical_block_size*size
        diskinf['blocksize'] = logical_block_size

        remaininfo = self.client.bash('parted dev/{} print '.format(diskname)).get().stdout
        remaininfo_lines = remaininfo.splitlines()
        for line in remaininfo_lines:
            if 'Partition Table' in line:
                diskinf['table'] = str(line[line.find(':')+2:])
        return diskinf

    def test001_create_list_delete_btrfs(self):
        """ g8os-008
        *Test case for creating, listing and monitoring btrfs*

        **Test Scenario:**
        #. Setup two loop devices to be used by btrfs
        #. Create Btrfs file system (Bfs1), should succeed
        #. Mount the btrfs filesystem (Bfs1)
        #. List Btrfs file system, should find the file system (Bfs1)
        #. Get Info for the btrfs file system (Bfs1)
        #. Add new loop (LD1) device, should succeed
        #. Remove the loop device (LD1), should succeed
        #. Remove all loop devices
        #. List the btrfs filesystem, Bfs1 shouldn't be there
        """

        self.lg('{} STARTED'.format(self._testID))

        self.create_btrfs()

        self.lg('List Btrfs file system, should find the file system (Bfs1)')
        btr_list = self.client.btrfs.list()
        btr = [i for i in btr_list if i['label'] == self.label]
        self.assertNotEqual(btr, [])

        self.lg('Get Info for the btrfs file system (Bfs1)')
        rs = self.client.btrfs.info(self.mount_point)
        self.assertEqual(rs['label'], self.label)
        self.assertEqual(rs['total_devices'], btr[0]['total_devices'])

        self.lg('Add new loop (LD1) device')
        loop_dev_list2 = self.setup_loop_devices(['bd2'], '500M')
        self.client.btrfs.device_add(self.mount_point, loop_dev_list2[0])
        rs = self.client.btrfs.info(self.mount_point)
        self.assertEqual(rs['total_devices'], 3)

        self.lg('Remove the loop device (LD1)')
        self.client.btrfs.device_remove(self.mount_point, loop_dev_list2[0])
        rs = self.client.btrfs.info(self.mount_point)
        self.assertEqual(rs['total_devices'], 2)

        self.destroy_btrfs()

        self.lg("List the btrfs filesystems , Bfs1 shouldn't be there")
        btr_list = self.client.btrfs.list()
        btr = [i for i in btr_list if i['label'] == self.label]
        self.assertEqual(btr, [])
        self.client.bash('rm -rf {}'.format(self.mount_point))

        self.lg('{} ENDED'.format(self._testID))

    def test002_subvolumes_btrfs(self):
        """ g8os-016
        *Test case for creating, listing and deleting btrfs subvolumes*

        **Test Scenario:**
        #. Create Btrfs file system, should succeed
        #. Create btrfs subvolume (SV1), should succeed
        #. List btrfs subvolumes giving any btrfs path, SV1 should be found
        #. List btrfs subvolumes giving non btrfs path, should fail
        #. Create btrfs subvolume (SV2) inside (SV1), should succeed
        #. Delete SV1, should fail as it has SV1 inside it
        #. Delete SV2 then SV1, should succeed
        #. List btrfs subvolumes, should return nothing
        #. Delete SV1, should Fail
        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create Btrfs file system, should succeed')
        self.create_btrfs()

        self.lg('Create btrfs subvolume (SV1), should succeed')
        sv1 = self.rand_str()
        sv1_path = '{}/{}'.format(self.mount_point, sv1)
        self.client.btrfs.subvol_create(sv1_path)

        self.lg('List btrfs subvolumes, SV1 should be found')
        sub_list = self.client.btrfs.subvol_list(self.mount_point)
        self.assertEqual(sub_list[0]['Path'], sv1)
        self.assertEqual(len(sub_list), 1)

        self.lg('List btrfs subvolumes giving non btrfs path, should fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.subvol_list('/mnt')

        self.lg('Create btrfs subvolume (SV2) inside (SV1), should succeed')
        sv2 = self.rand_str()
        sv2_path = '{}/{}'.format(sv1_path, sv2)
        self.client.btrfs.subvol_create(sv2_path)

        self.lg('Delete SV1, should fail as it has SV1 inside it')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.subvol_delete(sv1_path)

        self.lg('Delete SV2 then SV1, should succeed')
        self.client.btrfs.subvol_delete(sv2_path)
        self.client.btrfs.subvol_delete(sv1_path)

        self.lg('List btrfs subvolumes, should return nothing')
        sub_list = self.client.btrfs.subvol_list(self.mount_point)
        self.assertIsNone(sub_list)

        self.lg('Delete SV1, should Fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.subvol_delete(sv1_path)
        self.destroy_btrfs()

        self.lg('{} ENDED'.format(self._testID))

    def test003_snapshots_btrfs(self):
        """ g8os-0017
        *Test case for creating, listing and deleting btrfs snapshots*

        **Test Scenario:**
        #. Create Btrfs file system, should succeed
        #. Create btrfs snapshot (SN1), should succeed
        #. List btrfs subvolumes giving any btrfs path, SN1 should be found
        #. Create btrfs snapshot (SN2) inside (SN1) with read only, should succeed
        #. Create btrfs snapshot (SN3) inside (SN2), should fail
        #. Delete btrfs subvolume SN2 then SN1, should succeed
        #. List btrfs subvolumes, should return nothing
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create Btrfs file system, should succeed')
        self.create_btrfs()

        self.lg('Create btrfs snapshot (SN1), should succeed')
        sn1 = self.rand_str()
        sn1_path = '{}/{}'.format(self.mount_point, sn1)
        self.client.btrfs.subvol_snapshot(self.mount_point, sn1_path)

        self.lg('List btrfs subvolumes giving any btrfs path, SN1 should be found')
        sub_list = self.client.btrfs.subvol_list(self.mount_point)
        self.assertEqual(sub_list[0]['Path'], sn1)
        self.assertEqual(len(sub_list), 1)

        self.lg('Create btrfs snapshot (SN2) inside (SN1) with read only, should succeed')
        sn2 = self.rand_str()
        sn2_path = '{}/{}'.format(sn1_path, sn2)
        self.client.btrfs.subvol_snapshot(sn1_path, sn2_path, read_only=True)

        self.lg('Create btrfs snapshot (SN3) inside (SN2), should fail')
        sn3 = self.rand_str()
        sn3_path = '{}/{}'.format(sn2_path, sn3)
        with self.assertRaises(RuntimeError):
            self.client.btrfs.subvol_snapshot(sn2_path, sn3_path)

        self.lg('Delete btrfs subvolume SN2 then SN1, should succeed')
        self.client.btrfs.subvol_delete(sn2_path)
        self.client.btrfs.subvol_delete(sn1_path)

        self.lg('List btrfs subvolumes, should return nothing')
        sub_list = self.client.btrfs.subvol_list(self.mount_point)
        self.assertIsNone(sub_list)

        self.destroy_btrfs()

        self.lg('{} ENDED'.format(self._testID))

    def test004_disk_get_info(self):
        """ g8os-020

        *Test case for checking on the disks information*

        **Test Scenario:**

        #. Get the disks name  using disk list
        #. Get disk info using bash
        #. Get disk info using g8os disk info
        #. Compare g8os results to that of the bash results, should be the same

        """
        self.lg('{} STARTED'.format(self._testID))

        self.lg('Get the disks names  using disk list')
        disks_names = []
        disks = self.client.disk.list()
        for disk in disks['blockdevices']:
            disks_names.append(disk['name'])

        for disk in disks_names:

            self.lg('Get disk {} info  using g8os disk info  '.format(disk))
            g8os_disk_info = self.client.disk.getinfo(disk)
            keys = g8os_disk_info.keys()

            self.lg('Get disk {} info  using bash '.format(disk))
            bash_disk_info = self.bash_disk_info(keys, disk)

            self.lg('compare g8os results to disk{} of the bash results, should be the same '.format(disk))
            for key in g8os_disk_info.keys():
                if key in bash_disk_info.keys():
                    self.assertEqual(g8os_disk_info[key], bash_disk_info[key],'different in key {} for disk{} '.format(key,disk))

        self.lg('{} ENDED'.format(self._testID))

    def test005_disk_mount_and_unmount(self):
        """ g8os-021

        *Test case for test mount disk and unmount *

        **Test Scenario:**

        #. create loop device and put file system on it
        #. Mount disk using g8os disk mount.
        #. Get disk info , the mounted disk should be there.
        #. Try mount it again , should fail.
        #. unmount the disk, shouldn't be found in the disks list

        """
        self.lg('{} STARTED'.format(self._testID))
        filename = [self.rand_str()]
        label = self.rand_str()
        mount_point = '/mnt/{}'.format(self.rand_str())
        self.lg('Mount disk using g8os disk mount.')
        loop_dev_list = self.setup_loop_devices(filename, '500M', deattach=True)

        self.client.btrfs.create(label, loop_dev_list)

        self.lg('Mount disk using g8os disk mount')
        self.client.bash('mkdir -p {}'.format(mount_point))
        self.client.disk.mount(loop_dev_list[0], mount_point,[""])

        self.lg('Get disk info , the mounted disk should be there.')
        disks = self.client.bash(' lsblk -n -io NAME ').get().stdout
        disks = disks.splitlines()
        result = [disk in loop_dev_list[0] for disk in disks]
        self.assertTrue(True in result)

        self.lg('Try mount it again , should fail')
        with self.assertRaises(RuntimeError):
            self.client.disk.mount(loop_dev_list[0], mount_point, [""])

        self.lg('unmount the disk, shouldn\'t be found in the disks list')

        self.client.disk.umount(loop_dev_list[0])
        disks = self.client.bash(' lsblk -n -io NAME ').get().stdout
        disks = disks.splitlines()
        result = [disk in loop_dev_list[0] for disk in disks]
        self.assertFalse(True in result)

        self.lg('{} ENDED'.format(self._testID))

    def test006_disk_partitions(self):

        """ g8os-022

        *Test case for test creating Partitions in disk *

        **Test Scenario:**

        #. create loop device and put file system on it
        #. Make partition for disk before making partition table for it , should fail.
        #. Make a partition table for this disk, should succeed.
        #. Make 2 partition for disk with 50% space of disk , should succeed.
        #. check disk  exist in disk list with 2 partition .
        #. Make partition for this disk again  , should fail.
        #. Remove all partitions for this disk ,should succeed.

        """
        self.lg('{} STARTED'.format(self._testID))
        filename = [self.rand_str()]

        self.lg('create loop device and put file system on it')
        loop_dev_list = self.setup_loop_devices(filename, '500M', deattach=True)
        device_name = loop_dev_list[0]
        device_name = device_name[device_name.index('/')+5:]

        self.lg('Make partition for disk before making partition table for it , should fail.')
        with self.assertRaises(RuntimeError):
            self.client.disk.mkpart(device_name, '0', '50%')

        self.lg('Make a partition table for this disk, should succeed.')
        self.client.disk.mktable(device_name)

        self.lg('Make 2 partition for disk with 50% space of disk , should succeed.')
        self.client.disk.mkpart(device_name, '0', '50%')
        self.client.disk.mkpart(device_name, '50%', '100%')

        self.lg('check disk  exist in disk list with 2 partition ')
        disks = self.client.disk.list()
        for disk in disks['blockdevices']:
            if disk['name'] == device_name:
                self.assertEqual(len(disk['children']), 2)

        self.lg('Make partition for this disk again  , should fail.')
        with self.assertRaises(RuntimeError):
            self.client.disk.mkpart(device_name, '0', '50%')

        self.lg('Remove partition for this disk ,should succeed.')
        self.client.disk.rmpart(device_name, 1)
        self.client.disk.rmpart(device_name, 2)

        self.lg('check that  partitions for this disk removed ,should succeed.')
        disks = self.client.disk.list()
        for disk in disks['blockdevices']:
            if disk['name'] == device_name:
                self.assertTrue('children' not in disk.keys())

        self.lg('{} ENDED'.format(self._testID))

    def test007_extended_disk_partitions(self):

        """ g8os-025

        *Test case for test creating Partitions in disk *

        **Test Scenario:**

        #. Create loop device .
        #. Make a partition table with type msdos for this disk, should succeed.
        #. Check disk table type from disk info ,msdos type should be there .
        #. Make primary partition for disk with 50% space of disk.
        #. Make extended partition with remain disk space ,should succeed.
        #. Divide extended partition to logical partition ,should succeed .
        #. Check disk  exist in disk list with 2 partition.
        #. Mount partition 1 and 2 of disk  using g8os disk mount with rw option , should succeed.
        #. Get disk info, check the mounted point for partition.
        #. Remove mounted partition ,should fail
        #. Unmount the partition1, should succeed.

        """
        self.lg('{} STARTED'.format(self._testID))
        table_type = 'msdos'
        filename = [self.rand_str()]
        label = self.rand_str()
        mount_point_part1 = '/mnt/{}'.format(self.rand_str())
        mount_point_part2 = '/mnt/{}'.format(self.rand_str())
        self.lg('Create loop device ')
        loop_dev_list = self.setup_loop_devices(filename, '500M', deattach=True)
        device_name = loop_dev_list[0]
        device_name = device_name[device_name.index('/')+5:]

        self.lg('Make a partition table with  msdos type for this disk, should succeed.')
        self.client.disk.mktable(device_name, table_type)

        self.lg('Check disk table type from disk info ,msdos type should be there .')
        info = self.client.disk.getinfo(device_name)
        self.assertEqual(info['table'], table_type)

        self.lg('Make primary partition for disk with 50% space of disk.')
        self.client.disk.mkpart(device_name, '1', '50%')

        self.lg('Make extended partition with remain disk space ,should succeed.')
        self.client.disk.mkpart(device_name, '51%', '100%', 'extended')

        self.lg(' Divide extended partition to logical, should succeed.')
        self.client.disk.mkpart(device_name, '51%', '100%', 'logical')

        self.lg('check that disk exist in disk list with 2 partition ')
        disks = self.client.disk.list()
        for disk in disks['blockdevices']:
            if disk['name'] == device_name:
                self.assertEqual(len(disk['children']), 2)

        self.lg('Mount partition 1 of disk  using g8os disk mount with rw option , should succeed')
        self.client.btrfs.create(label, ['{}p1'.format(loop_dev_list[0])])
        self.client.bash('mkdir -p {}'.format(mount_point_part1))
        self.client.disk.mount('{}p1'.format(loop_dev_list[0]), mount_point_part1, ["rw"])

        self.lg('Mount partition 2 of disk  using g8os disk mount with rw option , should succeed')
        self.client.btrfs.create(label, ['{}p5'.format(loop_dev_list[0])])
        self.client.bash('mkdir -p {}'.format(mount_point_part2))
        self.client.disk.mount('{}p5'.format(loop_dev_list[0]), mount_point_part2, ["rw"])

        self.lg('Get disk info, check the mounted points for two partitions.')
        part1_name = '{}p1'.format(device_name)
        info = self.client.disk.getinfo(device_name)
        partitions = info['children']
        for part in partitions:
            if part['name'] == part1_name:
                self.assertEqual(str(part['mountpoint']), mount_point_part1)
            else:
                self.assertEqual(str(part['mountpoint']), mount_point_part2)

        self.lg('Remove mounted partion ,should fail')
        with self.assertRaises(RuntimeError):
            self.client.disk.rmpart('{}p5'.format(device_name),5)

        self.lg('Unmount the partitions, should succeed')
        self.client.disk.umount('{}p1'.format(loop_dev_list[0]))
        self.client.disk.umount('{}p5'.format(loop_dev_list[0]))

        self.lg('{} ENDED'.format(self._testID))

    def test008_quota_btrfs(self):
        """ g8os-026
        *Test case for btrfs quota*

        **Test Scenario:**
        #. Create Btrfs file system, should succeed
        #. Create btrfs subvolume (SV1), should succeed
        #. Apply btrfs quota for SV1 with limit (L1), should succeed
        #. Try to write file inside that directory exceeding L1, should fail
        #. Destroy this btrfs filesystem
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create Btrfs file system, should succeed')
        self.create_btrfs()

        self.lg('Create btrfs subvolume (SV1), should succeed')
        sv1 = self.rand_str()
        sv1_path = '{}/{}'.format(self.mount_point, sv1)
        self.client.btrfs.subvol_create(sv1_path)

        self.lg('Apply btrfs quota for SV1 with limit (L1), should succeed')
        L1 = '100M'
        self.client.btrfs.subvol_quota(sv1_path, L1)
        rs = self.client.bash('btrfs qgroup show -r {} | grep 100.00MiB'.format(sv1_path))
        self.assertEqual(rs.get().state, 'SUCCESS')
        self.lg('Try to write file inside that directory exceeding L1, should fail')
        rs = self.client.bash('cd {}; fallocate -l 200M {}'.format(sv1_path, self.rand_str()))
        self.assertEqual(rs.get().state, 'ERROR')
        self.assertEqual(rs.get().stderr, 'fallocate: fallocate failed: Disk quota exceeded\n')

        self.lg('Destroy this btrfs filesystem')
        self.destroy_btrfs()

        self.lg('{} ENDED'.format(self._testID))

    def test009_btrfs_add_remove_devices(self):
        """ g8os-034
        *Test case for adding and removing devices for btrfs *

        **Test Scenario:**
        #. Create Btrfs file system (Bfs1), should succeed
        #. Create Btrfs file system (Bfs2), should succeed
        #. Add device (D1) to the (Bfs1) using fake mount point, should fail
        #. Add device (D1) to the (Bfs1) mount point, should succeed
        #. Add (D1) again to the (Bfs1) mount point, should fail
        #. Remove device (D1) using (Bfs2) mount point, should fail
        #. Remove device (D1) using any fake device, should fail
        #. Remove device (D1) using fake mount point, should fail
        #. Remove device (D1) using (Bfs1) mount point, should succeed
        """

        self.lg('{} STARTED'.format(self._testID))

        self.lg('Create Btrfs file system, should succeed')
        self.create_btrfs()

        self.lg('Create Btrfs file system (Bfs1), should succeed')
        self.lg('Create Btrfs file system (Bfs2), should succeed')
        self.create_btrfs(second_btrfs=True)

        self.lg('Add device (D1) to the (Bfs1) using fake mount point, should fail')
        d1 = self.setup_loop_devices(['bd4'], '500M', deattach=False)[0]
        with self.assertRaises(RuntimeError):
            self.client.btrfs.device_add('/mnt/'.format(self.rand_str()), d1)

        self.lg('Add device (D1) to the (Bfs1) mount point, should succeed')
        self.client.btrfs.device_add(self.mount_point, d1)
        rs = self.client.bash('btrfs filesystem show | grep -o "loop0"')
        self.assertEqual(rs.get().stdout, 'loop0\n')
        self.assertEqual(rs.get().state, 'SUCCESS')

        self.lg('Add (D1) again to the (Bfs1) mount point, should fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.device_add(self.mount_point, d1)

        self.lg('Remove device (D1) using (Bfs2) mount point, should fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.device_remove(self.mount_point2, d1)

        self.lg('Remove device (D1) using any fake device, should fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.device_remove(self.mount_point, self.rand_str())

        self.lg('Remove device (D1) using fake mount point, should fail')
        with self.assertRaises(RuntimeError):
            self.client.btrfs.device_remove('/mnt/'.format(self.rand_str()), d1)

        self.lg('Remove device (D1) using (Bfs1) mount point, should succeed')
        self.client.btrfs.device_remove(self.mount_point, d1)
        rs = self.client.bash('btrfs filesystem show | grep -o {}'.format(d1))
        self.assertEqual(rs.get().stdout, '')
        self.assertEqual(rs.get().state, 'ERROR')

        self.lg('Destroy both (Bfs1) and (Bfs2)')
        self.destroy_btrfs()

        self.lg('{} ENDED'.format(self._testID))
