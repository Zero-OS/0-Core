import redis
import uuid
import json
import textwrap
import shlex
import base64
import signal
from g8core import typchk


DefaultTimeout = 10  # seconds


class Timeout(Exception):
    pass


class Return:
    def __init__(self, payload):
        self._payload = payload

    @property
    def payload(self):
        return self._payload

    @property
    def id(self):
        return self._payload['id']

    @property
    def data(self):
        """
        data returned by the process. Only available if process
        output data with the correct core level
        """
        return self._payload['data']

    @property
    def level(self):
        """data message level (if any)"""
        return self._payload['level']

    @property
    def starttime(self):
        """timestamp"""
        return self._payload['starttime'] / 1000

    @property
    def time(self):
        """execution time in millisecond"""
        return self._payload['time']

    @property
    def state(self):
        """
        exit state
        """
        return self._payload['state']

    @property
    def stdout(self):
        streams = self._payload.get('streams', None)
        return streams[0] if streams is not None and len(streams) >= 1 else ''

    @property
    def stderr(self):
        streams = self._payload.get('streams', None)
        return streams[1] if streams is not None and len(streams) >= 2 else ''

    def __repr__(self):
        return str(self)

    def __str__(self):
        tmpl = """\
        STATE: {state}
        STDOUT:
        {stdout}
        STDERR:
        {stderr}
        DATA:
        {data}
        """

        return textwrap.dedent(tmpl).format(state=self.state, stdout=self.stdout, stderr=self.stderr, data=self.data)


class Response:
    def __init__(self, client, id):
        self._client = client
        self._id = id
        self._queue = 'result:{}'.format(id)

    @property
    def id(self):
        return self._id

    def get(self, timeout=None):
        if timeout is None:
            timeout = self._client.timeout
        r = self._client._redis
        v = r.brpoplpush(self._queue, self._queue, timeout)
        if v is None:
            raise Timeout()
        payload = json.loads(v.decode())
        return Return(payload)


class InfoManager:
    def __init__(self, client):
        self._client = client

    def cpu(self):
        return self._client.json('info.cpu', {})

    def nic(self):
        return self._client.json('info.nic', {})

    def mem(self):
        return self._client.json('info.mem', {})

    def disk(self):
        return self._client.json('info.disk', {})

    def os(self):
        return self._client.json('info.os', {})


class ProcessManager:
    _process_chk = typchk.Checker({
        'id': str,
    })

    _kill_chk = typchk.Checker({
        'id': str,
        'signal': int,
    })

    def __init__(self, client):
        self._client = client

    def list(self, id=None):
        """
        List all running process (the ones that were started by the core itself)

        :param id: optional ID for the process to list
        """
        args = {'id': id}
        self._process_chk.check(args)
        return self._client.json('process.list', args)

    def kill(self, id, signal=signal.SIGTERM):
        """
        Kill a process with given id

        :WARNING: beware of what u kill, if u killed redis for example core0 or coreX won't be reachable


        :param id: process id to kill
        """
        args = {
            'id': id,
            'signal': int(signal),
        }
        self._kill_chk.check(args)
        return self._client.json('process.kill', args)

class FilesystemManager:
    def __init__(self, client):
        self._client = client

    def open(self, file, mode='r', perm=0o0644):
        """
        Opens a file on the node

        :param file: file path to open
        :param mode: open mode
        :param perm: file permission in octet form

        mode:
          'r' read only
          'w' write only
          '+' read/write
          'x' create if not exist
          'a' append
        :return: a file descriptor
        """
        args = {
            'file': file,
            'mode': mode,
            'perm': perm,
        }

        return self._client.json('filesystem.open', args)

    def exists(self, path):
        """
        Check if path exists

        :param path: path to file/dir
        :return: boolean
        """
        args = {
            'path': path,
        }

        return self._client.json('filesystem.exists', args)

    def list(self, path):
        """
        List all entries in directory
        :param path: path to dir
        :return: list of director entries
        """
        args = {
            'path': path,
        }

        return self._client.json('filesystem.list', args)

    def mkdir(self, path):
        """
        Make a new directory == mkdir -p path
        :param path: path to directory to create
        :return:
        """
        args = {
            'path': path,
        }

        return self._client.json('filesystem.mkdir', args)

    def remove(self, path):
        """
        Removes a path (recursively)

        :param path: path to remove
        :return:
        """
        args = {
            'path': path,
        }

        return self._client.json('filesystem.remove', args)

    def move(self, path, destination):
        """
        Move a path to destination

        :param path: source
        :param destination: destination
        :return:
        """
        args = {
            'path': path,
            'destination': destination,
        }

        return self._client.json('filesystem.move', args)

    def chmod(self, path, mode, recursive=False):
        """
        Change file/dir permission

        :param path: path of file/dir to change
        :param mode: octet mode
        :param recursive: apply chmod recursively
        :return:
        """
        args = {
            'path': path,
            'mode': mode,
            'recursive': recursive,
        }

        return self._client.json('filesystem.chmod', args)

    def chown(self, path, user, group, recursive=False):
        """
        Change file/dir owner

        :param path: path of file/dir
        :param user: user name
        :param group: group name
        :param recursive: apply chown recursively
        :return:
        """
        args = {
            'path': path,
            'user': user,
            'group': group,
            'recursive': recursive,
        }

        return self._client.json('filesystem.chown', args)

    def read(self, fd):
        """
        Read a block from the given file descriptor

        :param fd: file descriptor
        :return: bytes
        """
        args = {
            'fd': fd,
        }

        data = self._client.json('filesystem.read', args)
        return base64.decodebytes(data.encode())

    def write(self, fd, bytes):
        """
        Write a block of bytes to an open file descriptor (that is open with one of the writing modes

        :param fd: file descriptor
        :param bytes: bytes block to write
        :return:

        :note: don't overkill the node with large byte chunks, also for large file upload check the upload method.
        """
        args = {
            'fd': fd,
            'block': base64.encodebytes(bytes).decode(),
        }

        return self._client.json('filesystem.write', args)

    def close(self, fd):
        """
        Close file
        :param fd: file descriptor
        :return:
        """
        args = {
            'fd': fd,
        }

        return self._client.json('filesystem.close', args)

    def upload(self, remote, reader):
        """
        Uploads a file
        :param remote: remote file name
        :param reader: an object that implements the read(size) method (typically a file descriptor)
        :return:
        """

        fd = self.open(remote, 'wx')
        while True:
            chunk = reader.read(512*1024)
            if chunk == b'':
                break
            self.write(fd, chunk)
        self.close(fd)

    def download(self, remote, writer):
        """
        Downloads a file
        :param remote: remote file name
        :param writer: an object the implements the write(bytes) interface (typical a file descriptor)
        :return:
        """

        fd = self.open(remote)
        while True:
            chunk = self.read(fd)
            if chunk == b'':
                break
            writer.write(chunk)
        self.close(fd)

    def upload_file(self, remote, local):
        """
        Uploads a file
        :param remote: remote file name
        :param local: local file name
        :return:
        """
        file = open(local, 'rb')
        self.upload(remote, file)

    def download_file(self, remote, local):
        """
        Downloads a file
        :param remote: remote file name
        :param local: local file name
        :return:
        """
        file = open(local, 'wb')
        self.download(remote, file)

class BaseClient:
    _system_chk = typchk.Checker({
        'name': str,
        'args': [str],
        'dir': str,
        'stdin': str,
        'env': typchk.Or(typchk.Map(str, str), typchk.IsNone()),
    })

    _bash_chk = typchk.Checker({
        'stdin': str,
        'script': str,
    })

    def __init__(self, timeout=None):
        if timeout is None:
            self.timeout = DefaultTimeout
        self._info = InfoManager(self)
        self._process = ProcessManager(self)
        self._filesystem = FilesystemManager(self)

    @property
    def info(self):
        return self._info

    @property
    def process(self):
        return self._process

    @property
    def filesystem(self):
        return self._filesystem

    def raw(self, command, arguments):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :return: Response object
        """
        raise NotImplemented()

    def sync(self, command, arguments):
        """
        Same as self.raw except it do a response.get() waiting for the command execution to finish and reads the result

        :return: Result object
        """
        response = self.raw(command, arguments)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('invalid response: %s' % result.state, result)

        return result

    def json(self, command, arguments):
        """
        Same as self.sync except it assumes the returned result is json, and loads the payload of the return object

        :Return: Data
        """
        result = self.sync(command, arguments)
        if result.level != 20:
            raise RuntimeError('invalid result level, expecting json(20) got (%d)' % result.level)

        return json.loads(result.data)

    def ping(self):
        """
        Ping a node, checking for it's availability. a Ping should never fail unless the node is not reachable
        or not responsive.
        :return:
        """
        response = self.raw('core.ping', {})

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('invalid response: %s' % result.state)

        return json.loads(result.data)

    def system(self, command, dir='', stdin='', env=None):
        """
        Execute a command

        :param command:  command to execute (with its arguments) ex: `ls -l /root`
        :param dir: CWD of command
        :param stdin: Stdin data to feed to the command stdin
        :param env: dict with ENV variables that will be exported to the command
        :return:
        """
        parts = shlex.split(command)
        if len(parts) == 0:
            raise ValueError('invalid command')

        args = {
            'name': parts[0],
            'args': parts[1:],
            'dir': dir,
            'stdin': stdin,
            'env': env,
        }

        self._system_chk.check(args)
        response = self.raw(command='core.system', arguments=args)

        return response

    def bash(self, script, stdin=''):
        """
        Execute a bash script, or run a process inside a bash shell.

        :param script: Script to execute (can be multiline script)
        :param stdin: Stdin data to feed to the script
        :return:
        """
        args = {
            'script': script,
            'stdin': stdin,
        }
        self._bash_chk.check(args)
        response = self.raw(command='bash', arguments=args)

        return response


class ContainerClient(BaseClient):
    _raw_chk = typchk.Checker({
        'container': int,
        'command': {
            'command': str,
            'arguments': typchk.Any(),
        }
    })

    def __init__(self, client, container):
        super().__init__(client.timeout)

        self._client = client
        self._container = container

    def raw(self, command, arguments):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :return: Response object
        """
        args = {
            'container': self._container,
            'command': {
                'command': command,
                'arguments': arguments,
            },
        }

        #check input
        self._raw_chk.check(args)

        response = self._client.raw('corex.dispatch', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to dispatch command to container: %s' % result.data)

        cmd_id = json.loads(result.data)
        return self._client.response_for(cmd_id)


class ContainerManager:
    _create_chk = typchk.Checker({
        'root': str,
        'mount': typchk.Or(
            typchk.Map(str, str),
            typchk.IsNone()
        ),
        'host_network': bool,
        'network': {
            'zerotier': typchk.Or(
                str,
                typchk.IsNone()
            ),
            'bridge': typchk.Or(
                [typchk.Length((str,), 2)],
                typchk.IsNone()
            ),  # list of tuples each of length 2 or None
        },
        'port': typchk.Or(
            typchk.Map(int, int),
            typchk.IsNone()
        ),
        'hostname': typchk.Or(
            str,
            typchk.IsNone()
        ),
        'storage': typchk.Or(str, typchk.IsNone()),
    })

    _terminate_chk = typchk.Checker({
        'container': int
    })

    def __init__(self, client):
        self._client = client

    def create(self, root_url, mount=None, host_network=False, zerotier=None, bridge=None, port=None, hostname=None, storage=None):
        """
        Creater a new container with the given root plist, mount points and
        zerotier id, and connected to the given bridges
        :param root_url: The root filesystem plist
        :param mount: a dict with {host_source: container_target} mount points.
                      where host_source directory must exists.
                      host_source can be a url to a plist to mount.
        :param host_network: Specify if the container should share the same network stack as the host.
                             if True, container creation ignores both zerotier, bridge and ports arguments below. Not
                             giving errors if provided.
        :param zerotier: An optional zerotier netowrk ID to join
        :param bridge: A list of tuples as ('bridge_name': 'network_setup')
                       where :network_setup: can be one of the following
                       '' or 'none':
                            no IP is gonna be set on the link
                       'dhcp':
                            Run `udhcpc` on the container link, of course this will
                            only work if the `bridge` is created with `dnsmasq` networking
                       'CIDR':
                            Assign static IP to the link

                       Examples:
                        `bridge=[('br0', '127.0.0.100/24'), ('br1', 'dhcp')]`
        :param port: A dict of host_port: container_port pairs
                       Example:
                        `port={8080: 80, 7000:7000}`
        :param hostname: Specific hostname you want to give to the container.
                         if None it will automatically be set to core-x,
                         x beeing the ID of the container
        :param storage: A Url to the ardb storage to use to mount the root plist (or any other mount that requires g8fs)
                        if not provided, the default one from core0 configuration will be used.
        """


        args = {
            'root': root_url,
            'mount': mount,
            'host_network': host_network,
            'network': {
                'zerotier': zerotier,
                'bridge': bridge,
            },
            'port': port,
            'hostname': hostname,
            'storage': storage,
        }

        #validate input
        self._create_chk.check(args)

        response = self._client.raw('corex.create', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to create container %s' % result.data)

        return json.loads(result.data)

    def list(self):
        """
        List running containers
        :return: a dict with {container_id: <container info object>}
        """
        response = self._client.raw('corex.list', {})

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to list containers: %s' % result.data)

        return json.loads(result.data)

    def terminate(self, container):
        """
        Terminate a container given it's id

        :param container: container id
        :return:
        """
        args = {
            'container': container,
        }
        self._terminate_chk.check(args)
        response = self._client.raw('corex.terminate', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to terminate container: %s' % result.data)

    def client(self, container):
        """
        Return a client instance that is bound to that container.

        :param container: container id
        :return: Client object bound to the specified container id
        """
        return ContainerClient(self._client, container)


class BridgeManager:
    _bridge_create_chk = typchk.Checker({
        'name': str,
        'hwaddr': str,
        'network': {
            'mode': typchk.Or(typchk.Enum('static', 'dnsmasq'), typchk.IsNone()),
            'nat': bool,
            'settings': typchk.Map(str, str),
        }
    })

    _bridge_delete_chk = typchk.Checker({
        'name': str,
    })

    def __init__(self, client):
        self._client = client

    def create(self, name, hwaddr=None, network=None, nat=False, settings={}):
        """
        Create a bridge with the given name, hwaddr and networking setup
        :param name: name of the bridge (must be unique)
        :param hwaddr: MAC address of the bridge. If none, a one will be created for u
        :param network: Networking mode, options are none, static, and dnsmasq
        :param nat: If true, SNAT will be enabled on this bridge.
        :param settings: Networking setting, depending on the selected mode.
                        none:
                            no settings, bridge won't get any ip settings
                        static:
                            settings={'cidr': 'ip/net'}
                            bridge will get assigned the given IP address
                        dnsmasq:
                            settings={'cidr': 'ip/net', 'start': 'ip', 'end': 'ip'}
                            bridge will get assigned the ip in cidr
                            and each running container that is attached to this IP will get
                            IP from the start/end range. Netmask of the range is the netmask
                            part of the provided cidr.
                            if nat is true, SNAT rules will be automatically added in the firewall.
        """
        args = {
            'name': name,
            'hwaddr': hwaddr,
            'network': {
                'mode': network,
                'nat': nat,
                'settings': settings,
            }
        }

        self._bridge_create_chk.check(args)

        response = self._client.raw('bridge.create', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to create bridge %s' % result.data)

        return json.loads(result.data)

    def list(self):
        """
        List all available bridges
        :return: list of bridge names
        """
        response = self._client.raw('bridge.list', {})

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to list bridges: %s' % result.data)

        return json.loads(result.data)

    def delete(self, bridge):
        """
        Delete a bridge by name

        :param bridge: bridge name
        :return:
        """
        args = {
            'name': bridge,
        }

        self._bridge_delete_chk.check(args)

        response = self._client.raw('bridge.delete', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to list delete: %s' % result.data)


class DiskManager:
    _mktable_chk = typchk.Checker({
        'disk': str,
        'table_type': typchk.Enum('aix', 'amiga', 'bsd', 'dvh', 'gpt', 'mac', 'msdos', 'pc98', 'sun', 'loop')
    })

    _mkpart_chk = typchk.Checker({
        'disk': str,
        'start': typchk.Or(int, str),
        'end': typchk.Or(int, str),
        'part_type': typchk.Enum('primary', 'logical', 'extended'),
    })

    _getpart_chk = typchk.Checker({
        'disk': str,
        'part': str,
    })

    _rmpart_chk = typchk.Checker({
        'disk': str,
        'number': int,
    })

    _mount_chk = typchk.Checker({
        'options': str,
        'source': str,
        'target': str,
    })

    _umount_chk = typchk.Checker({
        'source': str,
    })

    def __init__(self, client):
        self._client = client

    def list(self):
        """
        List available block devices
        """
        response = self._client.raw('disk.list', {})

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to list disks: %s' % result.stderr)

        if result.level != 20:  # 20 is JSON output.
            raise RuntimeError('invalid response type from disk.list command')

        data = result.data.strip()
        if data:
            return json.loads(data)
        else:
            return {}

    def mktable(self, disk, table_type='gpt'):
        """
        Make partition table on block device.
        :param disk: Full device path like /dev/sda
        :param table_type: Partition table type as accepted by parted
        """
        args = {
            'disk': disk,
            'table_type': table_type,
        }

        self._mktable_chk.check(args)

        response = self._client.raw('disk.mktable', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to create table: %s' % result.stderr)

    def getinfo(self, disk, part=''):
        """
        Get more info about a disk or a disk partition

        :param disk: (sda, sdb, etc..)
        :param part: (sda1, sdb2, etc...)
        :return: a dict with {"blocksize", "start", "size", and "free" sections}
        """
        args = {
            "disk": disk,
            "part": part,
        }

        self._getpart_chk.check(args)

        response = self._client.raw('disk.getinfo', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to get info: %s' % result.stderr)

        if result.level != 20:  # 20 is JSON output.
            raise RuntimeError('invalid response type from disk.getinfo command')

        data = result.data.strip()
        if data:
            return json.loads(data)
        else:
            return {}

    def mkpart(self, disk, start, end, part_type='primary'):
        """
        Make partition on disk
        :param disk: device name (sda, sdb, etc...)
        :param start: partition start as accepted by parted mkpart
        :param end: partition end as accepted by parted mkpart
        :param part_type: partition type as accepted by parted mkpart
        """
        args = {
            'disk': disk,
            'start': start,
            'end': end,
            'part_type': part_type,
        }

        self._mkpart_chk.check(args)

        response = self._client.raw('disk.mkpart', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to create partition: %s' % result.stderr)

    def rmpart(self, disk, number):
        """
        Remove partion from disk
        :param disk: device name (sda, sdb, etc...)
        :param number: Partition number (starting from 1)
        """
        args = {
            'disk': disk,
            'number': number,
        }

        self._rmpart_chk.check(args)

        response = self._client.raw('disk.rmpart', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to remove partition: %s' % result.stderr)

    def mount(self, source, target, options=[]):
        """
        Mount partion on target
        :param source: Full partition path like /dev/sda1
        :param target: Mount point
        :param options: Optional mount options
        """

        if len(options) == 0:
            options = ['auto']

        args = {
            'options': ','.join(options),
            'source': source,
            'target': target,
        }

        self._mount_chk.check(args)
        response = self._client.raw('disk.mount', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to mount partition: %s' % result.stderr)

    def umount(self, source):
        """
        Unmount partion
        :param source: Full partition path like /dev/sda1
        """

        args = {
            'source': source,
        }
        self._umount_chk.check(args)

        response = self._client.raw('disk.umount', args)

        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to umount partition: %s' % result.stderr)


class BtrfsManager:
    _create_chk = typchk.Checker({
        'label': str,
        'metadata': typchk.Enum("raid0", "raid1", "raid5", "raid6", "raid10", "dup", "single", ""),
        'data': typchk.Enum("raid0", "raid1", "raid5", "raid6", "raid10", "dup", "single", ""),
        'devices': [str],
        'overwrite': bool,
    })

    _device_chk = typchk.Checker({
        'mountpoint': str,
        'devices': (str,),
    })

    _subvol_chk = typchk.Checker({
        'path': str,
    })

    _subvol_quota_chk = typchk.Checker({
        'path': str,
        'limit': str,
    })

    _subvol_snapshot_chk = typchk.Checker({
        'source': str,
        'destination': str,
        'read_only': bool,
    })

    def __init__(self, client):
        self._client = client

    def list(self):
        """
        List all btrfs filesystem
        """
        return self._client.json('btrfs.list', {})

    def info(self, mountpoint):
        """
        Get btrfs fs info
        """
        return self._client.json('btrfs.info', {'mountpoint': mountpoint})

    def create(self, label, devices, metadata_profile="", data_profile="", overwrite=False):
        """
        Create a btrfs filesystem with the given label, devices, and profiles
        :param label: name/label
        :param devices : array of devices (under /dev)
        :metadata_profile: raid0, raid1, raid5, raid6, raid10, dup or single
        :data_profile: same as metadata profile
        :overwrite: force creation of the filesystem. Overwrite any existing filesystem
        """
        args = {
            'label': label,
            'metadata': metadata_profile,
            'data': data_profile,
            'devices': devices,
            'overwrite': overwrite
        }

        self._create_chk.check(args)

        self._client.sync('btrfs.create', args)

    def device_add(self, mountpoint, *device):
        """
        Add one or more devices to btrfs filesystem mounted under `mountpoint`

        :param mountpoint: mount point of the btrfs system
        :param devices: one ore more devices to add
        :return:
        """
        if len(device) == 0:
            return

        args = {
            'mountpoint': mountpoint,
            'devices': device,
        }

        self._device_chk.check(args)

        self._client.sync('btrfs.device_add', args)

    def device_remove(self, mountpoint, *device):
        """
        Remove one or more devices from btrfs filesystem mounted under `mountpoint`

        :param mountpoint: mount point of the btrfs system
        :param devices: one ore more devices to remove
        :return:
        """
        if len(device) == 0:
            return

        args = {
            'mountpoint': mountpoint,
            'devices': device,
        }

        self._device_chk.check(args)

        self._client.raw('btrfs.device_remove', args)

    def subvol_create(self, path):
        """
        Create a btrfs subvolume in the specified path
        :param path: path to create
        """
        args = {
            'path': path
        }
        self._subvol_chk.check(args)
        self._client.sync('btrfs.subvol_create', args)

    def subvol_list(self, path):
        """
        List a btrfs subvolume in the specified path
        :param path: path to be listed
        """
        return self._client.json('btrfs.subvol_list', {
            'path': path
        })

    def subvol_delete(self, path):
        """
        Delete a btrfs subvolume in the specified path
        :param path: path to delete
        """
        args = {
            'path': path
        }

        self._subvol_chk.check(args)

        self._client.sync('btrfs.subvol_delete', args)

    def subvol_quota(self, path, limit):
        """
        Apply a quota to a btrfs subvolume in the specified path
        :param path:  path to delete
        :param limit: the limit to Apply
        """
        args = {
            'path': path,
            'limit': limit,
        }

        self._subvol_quota_chk.check(args)

        self._client.sync('btrfs.subvol_quota', args)

    def subvol_snapshot(self, source, destination, read_only=False):
        """
        Take a snapshot

        :param source: source path of subvol
        :param destination: destination path of snapshot
        :param read_only: Set read-only on the snapshot
        :return:
        """

        args = {
            "source": source,
            "destination": destination,
            "read_only": read_only,
        }

        self._subvol_snapshot_chk.check(args)
        self._client.sync('btrfs.subvol_snapshot', args)


class ZerotierManager:
    _network_chk = typchk.Checker({
        'network': str,
    })

    def __init__(self, client):
        self._client = client

    def join(self, network):
        """
        Join a zerotier network

        :param network: network id to join
        :return:
        """
        args = {'network': network}
        self._network_chk.check(args)
        response = self._client.raw('zerotier.join', args)
        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to join zerotier network: %s', result.stderr)

    def leave(self, network):
        """
        Leave a zerotier network

        :param network: network id to leave
        :return:
        """
        args = {'network': network}
        self._network_chk.check(args)
        response = self._client.raw('zerotier.leave', args)
        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to leave zerotier network: %s', result.stderr)

    def list(self):
        """
        List joined zerotier networks

        :return: list of joined networks with their info
        """
        response = self._client.raw('zerotier.list', {})
        result = response.get()

        if result.state != 'SUCCESS':
            raise RuntimeError('failed to join zerotier network: %s', result.stderr)

        data = result.data.strip()
        if data == '':
            return []

        return json.loads(data)


class KvmManager:
    _create_chk = typchk.Checker({
        'name': str,
        'media': [{
            'type': typchk.Or(
                typchk.Enum('disk', 'cdrom'),
                typchk.Missing()
            ),
            'url': str,
        }],
        'cpu': int,
        'memory': int,
        'bridge': typchk.Or([str], typchk.IsNone()),
        'port': typchk.Or(
            typchk.Map(int, int),
            typchk.IsNone()
        ),
    })

    _destroy_chk = typchk.Checker({
        'name': str,
    })

    def __init__(self, client):
        self._client = client

    def create(self, name, media, cpu=2, memory=512, port=None, bridge=None):
        """

        :param name: Name of the kvm domain
        :param media: array of media objects to attach to the machine, where the first object is the boot device
                      each media object is a dict of {url, and type} where type can be one of 'disk', or 'cdrom', or empty (default to disk)
                      example: [{'url': 'nbd+unix:///test?socket=/tmp/ndb.socket'}, {'type': 'cdrom': '/somefile.iso'}
        :param cpu: number of vcpu cores
        :param memory: memory in MiB
        :param port: A dict of host_port: container_port pairs
                       Example:
                        `port={8080: 80, 7000:7000}`
        :param bridge: array of extra bridges to connect the domain with. the bridges must exist on the host
                       By default, vm is automatically added to a default bridge.
        :return:
        """
        args = {
            'name': name,
            'media': media,
            'cpu': cpu,
            'memory': memory,
            'bridge': bridge,
            'port': port,
        }
        self._create_chk.check(args)

        self._client.sync('kvm.create', args)

    def destroy(self, name):
        """
        Destroy a kvm domain by name
        :param name: name of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'name': name,
        }
        self._destroy_chk.check(args)

        self._client.sync('kvm.destroy', args)

    def list(self):
        """
        List configured domains

        :return:
        """
        return self._client.json('kvm.list', {})


class Experimental:
    def __init__(self, client):
        self._kvm = KvmManager(client)

    @property
    def kvm(self):
        return self._kvm


class Client(BaseClient):
    def __init__(self, host, port=6379, password="", db=0, timeout=None):
        super().__init__(timeout=timeout)

        self._redis = redis.Redis(host=host, port=port, password=password, db=db)
        self._container_manager = ContainerManager(self)
        self._bridge_manager = BridgeManager(self)
        self._disk_manager = DiskManager(self)
        self._btrfs_manager = BtrfsManager(self)
        self._zerotier = ZerotierManager(self)
        self._experimntal = Experimental(self)

    @property
    def experimental(self):
        return self._experimntal

    @property
    def container(self):
        return self._container_manager

    @property
    def bridge(self):
        return self._bridge_manager

    @property
    def disk(self):
        return self._disk_manager

    @property
    def btrfs(self):
        return self._btrfs_manager

    @property
    def zerotier(self):
        return self._zerotier

    def raw(self, command, arguments):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :return: Response object
        """
        id = str(uuid.uuid4())

        payload = {
            'id': id,
            'command': command,
            'arguments': arguments,
        }

        self._redis.rpush('core:default', json.dumps(payload))

        return Response(self, id)

    def response_for(self, id):
        return Response(self, id)
