import redis
import uuid
import json
import textwrap
import shlex
import base64
import signal
import socket
import logging
import time
import sys
from . import typchk


DefaultTimeout = 10  # seconds

logger = logging.getLogger('g8core')


class JobNotFoundError(Exception):
    pass


class ResultError(RuntimeError):
    def __init__(self, msg, code=0):
        super().__init__(msg)
        self._message = msg
        self._code = code

    @property
    def code(self):
        return self._code

    @property
    def message(self):
        return self._message


class Return:

    def __init__(self, payload):
        self._payload = payload

    @property
    def payload(self):
        """
        Raw return object data
        :return: dict
        """
        return self._payload

    @property
    def id(self):
        """
        Job ID
        :return: string
        """
        return self._payload['id']

    @property
    def data(self):
        """
        Data returned by the process. Only available if process
        output data with the correct core level

        For example, if a job returns a json object the self.level will be 20 and the data will contain the serialized
        json object, other levels exists for yaml, toml, etc... it really depends on the running job
        return: python primitive (str, number, dict or array)
        """
        return self._payload['data']

    @property
    def level(self):
        """
        Data message level (if any)
        """
        return self._payload['level']

    @property
    def starttime(self):
        """
        Starttime as a timestamp
        """
        return self._payload['starttime'] / 1000

    @property
    def time(self):
        """
        Execution time in millisecond
        """
        return self._payload['time']

    @property
    def state(self):
        """
        Exit state
        :return: str one of [SUCCESS, ERROR, KILLED, TIMEOUT, UNKNOWN_CMD, DUPLICATE_ID]
        """
        return self._payload['state']

    @property
    def stdout(self):
        """
        The job stdout
        :return: string or None
        """
        streams = self._payload.get('streams', None)
        return streams[0] if streams is not None and len(streams) >= 1 else ''

    @property
    def stderr(self):
        """
        The job stderr
        :return: string or None
        """
        streams = self._payload.get('streams', None)
        return streams[1] if streams is not None and len(streams) >= 2 else ''

    @property
    def code(self):
        """
        Exit code of the job, this can be either one of the http codes, of (if the value > 1000)
        is the exit code of the underlaying process
        if code > 1000:
            exit_code = code - 1000

        """
        return self._payload.get('code', 500)

    def __repr__(self):
        return str(self)

    def __str__(self):
        tmpl = """\
        STATE: {code} {state}
        STDOUT:
        {stdout}
        STDERR:
        {stderr}
        DATA:
        {data}
        """

        return textwrap.dedent(tmpl).format(code=self.code, state=self.state, stdout=self.stdout, stderr=self.stderr, data=self.data)


class Response:
    def __init__(self, client, id):
        self._client = client
        self._id = id
        self._queue = 'result:{}'.format(id)

    @property
    def id(self):
        """
        Job ID
        :return: string
        """
        return self._id

    @property
    def exists(self):
        """
        Returns true if the job is still running or zero-os still knows about this job ID

        After a job is finished, a job remains on zero-os for max of 5min where you still can read the job result
        after the 5 min is gone, the job result is no more fetchable
        :return: bool
        """
        r = self._client._redis
        flag = '{}:flag'.format(self._queue)
        return bool(r.execute_command('LKEYEXISTS', flag))

    @property
    def running(self):
        """
        Returns true if job still in running state
        :return:
        """
        r = self._client._redis
        flag = '{}:flag'.format(self._queue)
        if bool(r.execute_command('LKEYEXISTS', flag)):
            return r.execute_command('LTTL', flag) == -1

        return False

    def stream(self, callback=None):
        """
        Runtime copy of job messages. This required the 'stream` flag to be set to True otherwise it will
        not be able to copy any output, while it will block until the process exits.

        :note: This function will block until it reaches end of stream or the process is no longer running.

        :param callback: callback method that will get called for each received message
                         callback accepts 3 arguments
                         - level int: the log message levels, refer to the docs for available levels
                                      and their meanings
                         - message str: the actual output message
                         - flags int: flags associated with this message
                                      - 0x2 means EOF with success exit status
                                      - 0x4 means EOF with error

                                      for example (eof = flag & 0x6) eof will be true for last message u will ever
                                      receive on this callback.

                         Note: if callback is none, a default callback will be used that prints output on stdout/stderr
                         based on level.
        :return: None
        """
        if callback is None:
            callback = Response.__default

        if not callable(callback):
            raise Exception('callback must be callable')

        queue = 'stream:%s' % self.id
        r = self._client._redis

        # we can terminate quickly by checking if the process is not running and it has no queued output.
        if not self.running and r.llen(queue) == 0:
            return

        while True:
            data = r.blpop(queue, 10)
            if data is None:
                if not self.running:
                    break
                continue
            _, body = data
            payload = json.loads(body.decode())
            message = payload['message']
            line = message['message']
            meta = message['meta']
            callback(meta >> 16, line, meta & 0xff)

            if meta & 0x6 != 0:
                break

    @staticmethod
    def __default(level, line, meta):
        w = sys.stdout if level == 1 else sys.stderr
        w.write(line)
        w.write('\n')

    def get(self, timeout=None):
        """
        Waits for a job to finish (max of given timeout seconds) and return job results. When a job exits get() will
        keep returning the same result until zero-os doesn't remember the job anymore (self.exists == False)

        :notes: the timeout here is a client side timeout, it's different than the timeout given to the job on start
        (like in system method) witch will cause the job to be killed if it exceeded this timeout.

        :param timeout: max time to wait for the job to finish in seconds
        :return: Return object
        """
        if timeout is None:
            timeout = self._client.timeout
        r = self._client._redis
        start = time.time()
        maxwait = timeout
        while maxwait > 0:
            if not self.exists:
                raise JobNotFoundError(self.id)
            v = r.brpoplpush(self._queue, self._queue, min(maxwait, 10))
            if v is not None:
                payload = json.loads(v.decode())
                r = Return(payload)
                logger.debug('%s << %s, stdout="%s", stderr="%s", data="%s"',
                             self._id, r.state, r.stdout, r.stderr, r.data[:1000])
                return r
            logger.debug('%s still waiting (%ss)', self._id, int(time.time() - start))
            maxwait -= 10
        raise TimeoutError()


class JSONResponse(Response):
    def __init__(self, response):
        super().__init__(response._client, response.id)

    def get(self, timeout=None):
        """
        Get response as json, will fail if the job doesn't return a valid json response

        :param timeout: client side timeout in seconds
        :return: int
        """
        result = super().get(timeout)
        if result.state != 'SUCCESS':
            raise ResultError(result.data, result.code)
        if result.level != 20:
            raise ResultError('not a json response: %d' % result.level, 406)

        return json.loads(result.data)


class InfoManager:

    def __init__(self, client):
        self._client = client

    def cpu(self):
        """
        CPU information
        :return:
        """
        return self._client.json('info.cpu', {})

    def nic(self):
        """
        Return (physical) network devices information including IPs
        :return:
        """
        return self._client.json('info.nic', {})

    def mem(self):
        """
        Memory information
        :return:
        """
        return self._client.json('info.mem', {})

    def disk(self):
        """
        Disk information
        :return:
        """
        return self._client.json('info.disk', {})

    def os(self):
        """
        Operating system info
        :return:
        """
        return self._client.json('info.os', {})

    def port(self):
        """
        Return information about open ports on the system (similar to netstat)
        :return:
        """
        return self._client.json('info.port', {})

    def version(self):
        """
        Return OS version
        :return:
        """
        return self._client.json('info.version', {})


class JobManager:
    _job_chk = typchk.Checker({
        'id': typchk.Or(str, typchk.IsNone()),
    })

    _kill_chk = typchk.Checker({
        'id': str,
        'signal': int,
    })

    def __init__(self, client):
        self._client = client

    def list(self, id=None):
        """
        List all running jobs

        :param id: optional ID for the job to list
        """
        args = {'id': id}
        self._job_chk.check(args)
        return self._client.json('job.list', args)

    def kill(self, id, signal=signal.SIGTERM):
        """
        Kill a job with given id

        :WARNING: beware of what u kill, if u killed redis for example core0 or coreX won't be reachable

        :param id: job id to kill
        """
        args = {
            'id': id,
            'signal': int(signal),
        }
        self._kill_chk.check(args)
        return self._client.json('job.kill', args)


class ProcessManager:
    _process_chk = typchk.Checker({
        'pid': typchk.Or(int, typchk.IsNone()),
    })

    _kill_chk = typchk.Checker({
        'pid': int,
        'signal': int,
    })

    def __init__(self, client):
        self._client = client

    def list(self, id=None):
        """
        List all running processes

        :param id: optional PID for the process to list
        """
        args = {'pid': id}
        self._process_chk.check(args)
        return self._client.json('process.list', args)

    def kill(self, pid, signal=signal.SIGTERM):
        """
        Kill a process with given pid

        :WARNING: beware of what u kill, if u killed redis for example core0 or coreX won't be reachable

        :param pid: PID to kill
        """
        args = {
            'pid': pid,
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
          'w' write only (truncate)
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

        fd = self.open(remote, 'w')
        while True:
            chunk = reader.read(512 * 1024)
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
        try:
            self.upload(remote, file)
        finally:
            file.close()

    def download_file(self, remote, local):
        """
        Downloads a file
        :param remote: remote file name
        :param local: local file name
        :return:
        """
        file = open(local, 'wb')
        try:
            self.download(remote, file)
        finally:
            file.close()


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
        else:
            self.timeout = timeout
        self._info = InfoManager(self)
        self._job = JobManager(self)
        self._process = ProcessManager(self)
        self._filesystem = FilesystemManager(self)
        self._ip = IPManager(self)

    @property
    def info(self):
        """
        info manager
        :return:
        """
        return self._info

    @property
    def job(self):
        """
        job manager
        :return:
        """
        return self._job

    @property
    def process(self):
        """
        process manager
        :return:
        """
        return self._process

    @property
    def filesystem(self):
        """
        filesystem manager
        :return:
        """
        return self._filesystem

    @property
    def ip(self):
        """
        ip manager
        :return:
        """
        return self._ip

    def raw(self, command, arguments, queue=None, max_time=None, stream=False, tags=None, id=None):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :param queue: command queue (commands on the same queue are executed sequentially)
        :param max_time: kill job server side if it exceeded this amount of seconds
        :param stream: If True, process stdout and stderr are pushed to a special queue (stream:<id>) so
            client can stream output
        :param tags: job tags
        :param id: job id. Generated if not supplied
        :return: Response object
        """
        raise NotImplemented()

    def sync(self, command, arguments, tags=None, id=None):
        """
        Same as self.raw except it do a response.get() waiting for the command execution to finish and reads the result
        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :param tags: job tags
        :param id: job id. Generated if not supplied
        :return: Result object
        """
        response = self.raw(command, arguments, tags=tags, id=id)

        result = response.get()
        if result.state != 'SUCCESS':
            if not result.code:
                result.code = 500
            raise ResultError(msg='%s' % result.data, code=result.code)
                

        return result

    def json(self, command, arguments, tags=None, id=None):
        """
        Same as self.sync except it assumes the returned result is json, and loads the payload of the return object
        if the returned (data) is not of level (20) an error is raised.
        :Return: Data
        """
        result = self.sync(command, arguments, tags=tags, id=id)
        if result.level != 20:
            raise RuntimeError('invalid result level, expecting json(20) got (%d)' % result.level)

        return json.loads(result.data)

    def ping(self):
        """
        Ping a node, checking for it's availability. a Ping should never fail unless the node is not reachable
        or not responsive.
        :return:
        """
        return self.json('core.ping', {})

    def system(self, command, dir='', stdin='', env=None, queue=None, max_time=None, stream=False, tags=None, id=None):
        """
        Execute a command

        :param command:  command to execute (with its arguments) ex: `ls -l /root`
        :param dir: CWD of command
        :param stdin: Stdin data to feed to the command stdin
        :param env: dict with ENV variables that will be exported to the command
        :param id: job id. Auto generated if not defined.
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
        response = self.raw(command='core.system', arguments=args,
                            queue=queue, max_time=max_time, stream=stream, tags=tags, id=id)

        return response

    def bash(self, script, stdin='', queue=None, max_time=None, stream=False, tags=None, id=None):
        """
        Execute a bash script, or run a process inside a bash shell.

        :param script: Script to execute (can be multiline script)
        :param stdin: Stdin data to feed to the script
        :param id: job id. Auto generated if not defined.
        :return:
        """
        args = {
            'script': script,
            'stdin': stdin,
        }
        self._bash_chk.check(args)
        response = self.raw(command='bash', arguments=args,
                            queue=queue, max_time=max_time, stream=stream, tags=tags, id=id)

        return response

    def subscribe(self, job, id=None):
        """
        Subscribes to job logs. It return the subscribe Response object which you will need to call .stream() on
        to read the output stream of this job.

        Calling subscribe multiple times will cause different subscriptions on the same job, each subscription will
        have a copy of this job streams.

        Note: killing the subscription job will not affect this job, it will also not cause unsubscripe from this stream
        the subscriptions will die automatically once this job exits.

        example:
            job = client.system('long running job')
            subscription = client.subscribe(job.id)

            subscription.stream() # this will print directly on stdout/stderr check stream docs for more details.

        hint: u can give an optional id to the subscriber (otherwise a guid will be generate for you). You probably want
        to use this in case your job watcher died, so u can hook on the stream of the current subscriber instead of creating a new one

        example:
            job = client.system('long running job')
            subscription = client.subscribe(job.id, 'my-job-subscriber')

            subscription.stream()

            # process dies for any reason
            # on next start u can simply do

            subscription = client.response_for('my-job-subscriber')
            subscription.stream()


        :param job: the job ID to subscribe to
        :param id: the subscriber ID (optional)
        :return: the subscribe Job object
        """
        return self.raw('core.subscribe', {'id': job}, stream=True, id=id)


class ContainerClient(BaseClient):
    class ContainerZerotierManager:
        def __init__(self, client, container):
            self._container = container
            self._client = client

        def info(self):
            return self._client.json('corex.zerotier.info', {'container': self._container})

        def list(self):
            return self._client.json('corex.zerotier.list', {'container': self._container})

    _raw_chk = typchk.Checker({
        'container': int,
        'command': {
            'command': str,
            'arguments': typchk.Any(),
            'queue': typchk.Or(str, typchk.IsNone()),
            'max_time': typchk.Or(int, typchk.IsNone()),
            'stream': bool,
            'tags': typchk.Or([str], typchk.IsNone()),
            'id': typchk.Or(str, typchk.IsNone()),

        }
    })

    def __init__(self, client, container):
        super().__init__(client.timeout)

        self._client = client
        self._container = container
        self._zerotier = ContainerClient.ContainerZerotierManager(client, container)  # not (self) we use core0 client

    @property
    def container(self):
        """
        :return: container id
        """
        return self._container

    @property
    def zerotier(self):
        """
        information about zerotier id
        :return:
        """
        return self._zerotier

    def raw(self, command, arguments, queue=None, max_time=None, stream=False, tags=None, id=None):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :param queue: command queue (commands on the same queue are executed sequentially)
        :param max_time: kill job server side if it exceeded this amount of seconds
        :param stream: If True, process stdout and stderr are pushed to a special queue (stream:<id>) so
            client can stream output
        :param tags: job tags
        :param id: job id. Generated if not supplied
        :return: Response object
        """
        args = {
            'container': self._container,
            'command': {
                'command': command,
                'arguments': arguments,
                'queue': queue,
                'max_time': max_time,
                'stream': stream,
                'tags': tags,
                'id': id,
            },
        }

        # check input
        self._raw_chk.check(args)

        response = self._client.raw('corex.dispatch', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to dispatch command to container: %s' % result.data)

        cmd_id = json.loads(result.data)
        return self._client.response_for(cmd_id)


class ContainerManager:
    _nic = {
        'type': typchk.Enum('default', 'bridge', 'zerotier', 'vlan', 'vxlan'),
        'id': typchk.Or(str, typchk.Missing()),
        'name': typchk.Or(str, typchk.Missing()),
        'hwaddr': typchk.Or(str, typchk.Missing()),
        'config': typchk.Or(
            typchk.Missing(),
            {
                'dhcp': typchk.Or(bool, typchk.Missing()),
                'cidr': typchk.Or(str, typchk.Missing()),
                'gateway': typchk.Or(str, typchk.Missing()),
                'dns': typchk.Or([str], typchk.Missing()),
            }
        ),
        'monitor': typchk.Or(bool, typchk.Missing()),
    }

    _create_chk = typchk.Checker({
        'root': str,
        'mount': typchk.Or(
            typchk.Map(str, str),
            typchk.IsNone()
        ),
        'host_network': bool,
        'nics': [_nic],
        'port': typchk.Or(
            typchk.Map(int, int),
            typchk.IsNone()
        ),
        'privileged': bool,
        'hostname': typchk.Or(
            str,
            typchk.IsNone()
        ),
        'storage': typchk.Or(str, typchk.IsNone()),
        'name': typchk.Or(str, typchk.IsNone()),
        'identity': typchk.Or(str, typchk.IsNone()),
        'env': typchk.Or(typchk.IsNone(), typchk.Map(str, str))
    })

    _client_chk = typchk.Checker(
        typchk.Or(int, str)
    )

    _nic_add = typchk.Checker({
        'container': int,
        'nic': _nic,
    })

    _nic_remove = typchk.Checker({
        'container': int,
        'index': int,
    })

    DefaultNetworking = object()



    def __init__(self, client):
        self._client = client

    def create(self, root_url, mount=None, host_network=False, nics=DefaultNetworking, port=None, hostname=None, privileged=False, storage=None, name=None, tags=None, identity=None, env=None):
        """
        Creater a new container with the given root flist, mount points and
        zerotier id, and connected to the given bridges
        :param root_url: The root filesystem flist
        :param mount: a dict with {host_source: container_target} mount points.
                      where host_source directory must exists.
                      host_source can be a url to a flist to mount.
        :param host_network: Specify if the container should share the same network stack as the host.
                             if True, container creation ignores both zerotier, bridge and ports arguments below. Not
                             giving errors if provided.
        :param nics: Configure the attached nics to the container
                     each nic object is a dict of the format
                     {
                        'type': nic_type # default, bridge, zerotier, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
                        'id': id # depends on the type, bridge name, zerotier network id, the vlan tag or the vxlan id
                        'name': name of the nic inside the container (ignored in zerotier type)
                        'hwaddr': Mac address of nic.
                        'config': { # config is only honored for bridge, vlan, and vxlan types
                            'dhcp': bool,
                            'cidr': static_ip # ip/mask
                            'gateway': gateway
                            'dns': [dns]
                        }
                     }
        :param port: A dict of host_port: container_port pairs (only if default networking is enabled)
                       Example:
                        `port={8080: 80, 7000:7000}`
        :param hostname: Specific hostname you want to give to the container.
                         if None it will automatically be set to core-x,
                         x beeing the ID of the container
        :param privileged: If true, container runs in privileged mode.
        :param storage: A Url to the ardb storage to use to mount the root flist (or any other mount that requires g8fs)
                        if not provided, the default one from core0 configuration will be used.
        :param name: Optional name for the container
        :param identity: Container Zerotier identity, Only used if at least one of the nics is of type zerotier
        :param env: a dict with the environment variables needed to be set for the container
        """

        if nics == self.DefaultNetworking:
            nics = [{'type': 'default'}]
        elif nics is None:
            nics = []

        args = {
            'root': root_url,
            'mount': mount,
            'host_network': host_network,
            'nics': nics,
            'port': port,
            'hostname': hostname,
            'privileged': privileged,
            'storage': storage,
            'name': name,
            'identity': identity,
            'env': env
        }

        # validate input
        self._create_chk.check(args)

        response = self._client.raw('corex.create', args, tags=tags)

        return JSONResponse(response)

    def list(self):
        """
        List running containers
        :return: a dict with {container_id: <container info object>}
        """
        return self._client.json('corex.list', {})

    def find(self, *tags):
        """
        Find containers that matches set of tags
        :param tags:
        :return:
        """
        tags = list(map(str, tags))
        return self._client.json('corex.find', {'tags': tags})

    def terminate(self, container):
        """
        Terminate a container given it's id

        :param container: container id
        :return:
        """
        self._client_chk.check(container)
        args = {
            'container': int(container),
        }
        response = self._client.raw('corex.terminate', args)

        result = response.get()
        if result.state != 'SUCCESS':
            raise RuntimeError('failed to terminate container: %s' % result.data)

    def nic_add(self, container, nic):
        """
        Hot plug a nic into a container

        :param container: container ID
        :param nic: {
                        'type': nic_type # default, bridge, zerotier, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
                        'id': id # depends on the type, bridge name, zerotier network id, the vlan tag or the vxlan id
                        'name': name of the nic inside the container (ignored in zerotier type)
                        'hwaddr': Mac address of nic.
                        'config': { # config is only honored for bridge, vlan, and vxlan types
                            'dhcp': bool,
                            'cidr': static_ip # ip/mask
                            'gateway': gateway
                            'dns': [dns]
                        }
                     }
        :return:
        """
        args = {
            'container': container,
            'nic': nic
        }
        self._nic_add.check(args)

        return self._client.json('corex.nic-add', args)

    def nic_remove(self, container, index):
        """
        Hot unplug of nic from a container

        Note: removing a nic, doesn't remove the nic from the container info object, instead it sets it's state
        to `destroyed`.

        :param container: container ID
        :param index: index of the nic as returned in the container object info (as shown by container.list())
        :return:
        """
        args = {
            'container': container,
            'index': index
        }
        self._nic_remove.check(args)

        return self._client.json('corex.nic-remove', args)

    def client(self, container):
        """
        Return a client instance that is bound to that container.

        :param container: container id
        :return: Client object bound to the specified container id
        Return a ContainerResponse from container.create
        """

        self._client_chk.check(container)
        return ContainerClient(self._client, int(container))

    def backup(self, container, url):
        """
        Backup a container to the given restic url
        all restic urls are supported

        :param container:
        :param url: Url to restic repo
                examples
                (file:///path/to/restic/?password=<password>)

        :return: Json response to the backup job (do .get() to get the snapshot ID
        """

        args = {
            'container': container,
            'url': url,
        }

        return JSONResponse(self._client.raw('corex.backup', args))

    def restore(self, url, tags=None):
        """
        Full restore of a container backup. This restore method will recreate
        an exact copy of the backedup container (including same network setup, and other
        configurations as defined by the `create` method.

        To just restore the container data, and use new configuration, use the create method instead
        with the `root_url` set to `restic:<url>`

        :param url: Snapshot url, the snapshot ID is passed as a url fragment
                    examples:
                        `file:///path/to/restic/repo?password=<password>#<snapshot-id>`
        :param tags: this will always override the original container tags (even if not set)
        :return:
        """
        args = {
            'url': url,
        }

        return JSONResponse(self._client.raw('corex.restore', args, tags=tags))


class IPManager:
    class IPBridgeManager:
        def __init__(self, client):
            self._client = client

        def add(self, name, hwaddr=None):
            """
            Add bridge with given name and optional hardware address

            For more advanced bridge options please check the `bridge` manager.
            :param name: bridge name
            :param hwaddr: mac address (str)
            :return:
            """
            args = {
                'name': name,
                'hwaddr': hwaddr,
            }

            return self._client.json("ip.bridge.add", args)

        def delete(self, name):
            """
            Delete bridge with given name
            :param name: bridge name to delete
            :return:
            """
            args = {
                'name': name,
            }

            return self._client.json("ip.bridge.del", args)

        def addif(self, name, inf):
            """
            Add interface to bridge
            :param name: bridge name
            :param inf: interface name to add
            :return:
            """
            args = {
                'name': name,
                'inf': inf,
            }

            return self._client.json('ip.bridge.addif', args)

        def delif(self, name, inf):
            """
            Delete interface from bridge
            :param name: bridge name
            :param inf: interface to remove
            :return:
            """
            args = {
                'name': name,
                'inf': inf,
            }

            return self._client.json('ip.bridge.delif', args)

    class IPLinkManager:
        def __init__(self, client):
            self._client = client

        def up(self, link):
            """
            Set interface state to UP

            :param link: link/interface name
            :return:
            """
            args = {
                'name': link,
            }
            return self._client.json('ip.link.up', args)

        def down(self, link):
            """
            Set link/interface state to DOWN

            :param link: link/interface name
            :return:
            """
            args = {
                'name': link,
            }
            return self._client.json('ip.link.down', args)

        def name(self, link, name):
            """
            Rename link

            :param link: link to rename
            :param name: new name
            :return:
            """
            args = {
                'name': link,
                'new': name,
            }
            return self._client.json('ip.link.name', args)

        def list(self):
            return self._client.json('ip.link.list', {})

    class IPAddrManager:
        def __init__(self, client):
            self._client = client

        def add(self, link, ip):
            """
            Add IP to link

            :param link: link
            :param ip: ip address to add
            :return:
            """
            args = {
                'name': link,
                'ip': ip,
            }
            return self._client.json('ip.addr.add', args)

        def delete(self, link, ip):
            """
            Delete IP from link

            :param link: link
            :param ip: ip address to remove
            :return:
            """
            args = {
                'name': link,
                'ip': ip,
            }
            return self._client.json('ip.addr.del', args)

        def list(self, link):
            """
            List IPs of a link

            :param link: link name
            :return:
            """
            args = {
                'name': link,
            }
            return self._client.json('ip.addr.list', args)

    class IPRouteManager:
        def __init__(self, client):
            self._client = client

        def add(self, dev, dst, gw=None):
            """
            Add a route

            :param dev: device name
            :param dst: destination network
            :param gw: optional gateway
            :return:
            """
            args = {
                'dev': dev,
                'dst': dst,
                'gw': gw,
            }
            return self._client.json('ip.route.add', args)

        def delete(self, dev, dst, gw=None):
            """
            Delete a route

            :param dev: device name
            :param dst: destination network
            :param gw: optional gateway
            :return:
            """
            args = {
                'dev': dev,
                'dst': dst,
                'gw': gw,
            }
            return self._client.json('ip.route.del', args)

        def list(self):
            return self._client.json('ip.route.list', {})

    def __init__(self, client):
        self._client = client
        self._bridge = IPManager.IPBridgeManager(client)
        self._link = IPManager.IPLinkManager(client)
        self._addr = IPManager.IPAddrManager(client)
        self._route = IPManager.IPRouteManager(client)

    @property
    def bridge(self):
        """
        Bridge manager
        :return:
        """
        return self._bridge

    @property
    def link(self):
        """
        Link manager
        :return:
        """
        return self._link

    @property
    def addr(self):
        """
        Address manager
        :return:
        """
        return self._addr

    @property
    def route(self):
        """
        Route manager
        :return:
        """
        return self._route


class BridgeManager:
    _bridge_create_chk = typchk.Checker({
        'name': str,
        'hwaddr': typchk.Or(str, typchk.IsNone()),
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
        :param name: name of the bridge (must be unique), 15 characters or less, and not equal to "default".
        :param hwaddr: MAC address of the bridge. If none, a one will be created for u
        :param network: Networking mode, options are none, static, and dnsmasq
        :param nat: If true, SNAT will be enabled on this bridge. (IF and ONLY IF an IP is set on the bridge
                    via the settings, otherwise flag will be ignored) (the cidr attribute of either static, or dnsmasq modes)
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
        :param disk: device name (sda, sdb, etc...)
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
            raise RuntimeError('failed to get info: %s' % result.data)

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
            options = ['']

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
        'devices': typchk.Length([str], 1),
        'overwrite': bool,
    })

    _device_chk = typchk.Checker({
        'mountpoint': str,
        'devices': typchk.Length((str,), 1),
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
        :param devices : array of devices (/dev/sda1, etc...)
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

        self._client.sync('btrfs.device_remove', args)

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
        :param path:  path to apply the quota for (it has to be the path of the subvol)
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
        return self._client.json('zerotier.list', {})

    def info(self):
        """
        Display zerotier status info

        :return: dict of zerotier statusinfo
        """
        return self._client.json('zerotier.info', {})


class KvmManager:
    _iotune_dict = {
        'totalbytessecset': typchk.Or(bool, typchk.Missing()),
        'totalbytessec': typchk.Or(int, typchk.Missing()),
        'readbytessecset': typchk.Or(bool, typchk.Missing()),
        'readbytessec': typchk.Or(int, typchk.Missing()),
        'writebytessecset': typchk.Or(bool, typchk.Missing()),
        'writebytessec': typchk.Or(int, typchk.Missing()),
        'totaliopssecset': typchk.Or(bool, typchk.Missing()),
        'totaliopssec': typchk.Or(int, typchk.Missing()),
        'readiopssecset': typchk.Or(bool, typchk.Missing()),
        'readiopssec': typchk.Or(int, typchk.Missing()),
        'writeiopssecset': typchk.Or(bool, typchk.Missing()),
        'writeiopssec': typchk.Or(int, typchk.Missing()),
        'totalbytessecmaxset': typchk.Or(bool, typchk.Missing()),
        'totalbytessecmax': typchk.Or(int, typchk.Missing()),
        'readbytessecmaxset': typchk.Or(bool, typchk.Missing()),
        'readbytessecmax': typchk.Or(int, typchk.Missing()),
        'writebytessecmaxset': typchk.Or(bool, typchk.Missing()),
        'writebytessecmax': typchk.Or(int, typchk.Missing()),
        'totaliopssecmaxset': typchk.Or(bool, typchk.Missing()),
        'totaliopssecmax': typchk.Or(int, typchk.Missing()),
        'readiopssecmaxset': typchk.Or(bool, typchk.Missing()),
        'readiopssecmax': typchk.Or(int, typchk.Missing()),
        'writeiopssecmaxset': typchk.Or(bool, typchk.Missing()),
        'writeiopssecmax': typchk.Or(int, typchk.Missing()),
        'totalbytessecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'totalbytessecmaxlength': typchk.Or(int, typchk.Missing()),
        'readbytessecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'readbytessecmaxlength': typchk.Or(int, typchk.Missing()),
        'writebytessecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'writebytessecmaxlength': typchk.Or(int, typchk.Missing()),
        'totaliopssecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'totaliopssecmaxlength': typchk.Or(int, typchk.Missing()),
        'readiopssecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'readiopssecmaxlength': typchk.Or(int, typchk.Missing()),
        'writeiopssecmaxlengthset': typchk.Or(bool, typchk.Missing()),
        'writeiopssecmaxlength': typchk.Or(int, typchk.Missing()),
        'sizeiopssecset': typchk.Or(bool, typchk.Missing()),
        'sizeiopssec': typchk.Or(int, typchk.Missing()),
        'groupnameset': typchk.Or(bool, typchk.Missing()),
        'groupname': typchk.Or(str, typchk.Missing()),
    }
    _media_dict = {
        'type': typchk.Or(
            typchk.Enum('disk', 'cdrom'),
            typchk.Missing()
        ),
        'url': str,
        'iotune': typchk.Or(
            _iotune_dict,
            typchk.Missing()
        )
    }
    _create_chk = typchk.Checker({
        'name': str,
        'media': typchk.Length([_media_dict], 1),
        'cpu': int,
        'memory': int,
        'nics': [{
            'type': typchk.Enum('default', 'bridge', 'vxlan', 'vlan'),
            'id': typchk.Or(str, typchk.Missing()),
            'hwaddr': typchk.Or(str, typchk.Missing()),
        }],
        'port': typchk.Or(
            typchk.Map(int, int),
            typchk.IsNone()
        ),
    })

    _migrate_network_chk = typchk.Checker({
        'nics': [{
            'type': typchk.Enum('default', 'bridge', 'vxlan', 'vlan'),
            'id': typchk.Or(str, typchk.Missing()),
            'hwaddr': typchk.Or(str, typchk.Missing()),
        }],
        'port': typchk.Or(
            typchk.Map(int, int),
            typchk.IsNone()
        ),
        'uuid': str
    })

    _domain_action_chk = typchk.Checker({
        'uuid': str,
    })

    _man_disk_action_chk = typchk.Checker({
        'uuid': str,
        'media': _media_dict,
    })

    _man_nic_action_chk = typchk.Checker({
        'uuid': str,
        'type': typchk.Enum('default', 'bridge', 'vxlan', 'vlan'),
        'id': typchk.Or(str, typchk.IsNone()),
        'hwaddr': typchk.Or(str, typchk.IsNone()),
    })

    _migrate_action_chk = typchk.Checker({
        'uuid': str,
        'desturi': str,
    })

    _limit_disk_io_dict = {
        'uuid': str,
        'media': _media_dict,
    }

    _limit_disk_io_dict.update(_iotune_dict)

    _limit_disk_io_action_chk = typchk.Checker(_limit_disk_io_dict)

    def __init__(self, client):
        self._client = client

    def create(self, name, media, cpu=2, memory=512, nics=None, port=None, tags=None):
        """
        :param name: Name of the kvm domain
        :param media: array of media objects to attach to the machine, where the first object is the boot device
                      each media object is a dict of {url, type} where type can be one of 'disk', or 'cdrom', or empty (default to disk)
                      example: [{'url': 'nbd+unix:///test?socket=/tmp/ndb.socket'}, {'type': 'cdrom': '/somefile.iso'}
        :param cpu: number of vcpu cores
        :param memory: memory in MiB
        :param port: A dict of host_port: container_port pairs
                       Example:
                        `port={8080: 80, 7000:7000}`
                     Only supported if default network is used
        :param nics: Configure the attached nics to the container
                     each nic object is a dict of the format
                     {
                        'type': nic_type # default, bridge, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
                        'id': id # depends on the type, bridge name (bridge type) zerotier network id (zertier type), the vlan tag or the vxlan id
                     }
        :return: uuid of the virtual machine
        """

        if nics is None:
            nics = []

        args = {
            'name': name,
            'media': media,
            'cpu': cpu,
            'memory': memory,
            'nics': nics,
            'port': port,
        }
        self._create_chk.check(args)

        return self._client.sync('kvm.create', args, tags=tags)

    def prepare_migration_target(self, uuid, nics=None, port=None, tags=None):
        """
        :param name: Name of the kvm domain that will be migrated
        :param port: A dict of host_port: container_port pairs
                       Example:
                        `port={8080: 80, 7000:7000}`
                     Only supported if default network is used
        :param nics: Configure the attached nics to the container
                     each nic object is a dict of the format
                     {
                        'type': nic_type # default, bridge, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
                        'id': id # depends on the type, bridge name (bridge type) zerotier network id (zertier type), the vlan tag or the vxlan id
                     }
        :param uuid: uuid of machine to be migrated on old node
        :return:
        """

        if nics is None:
            nics = []

        args = {
            'nics': nics,
            'port': port,
            'uuid': uuid
        }
        self._migrate_network_chk.check(args)

        self._client.sync('kvm.prepare_migration_target', args, tags=tags)

    def destroy(self, uuid):
        """
        Destroy a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.destroy', args)

    def shutdown(self, uuid):
        """
        Shutdown a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.shutdown', args)

    def reboot(self, uuid):
        """
        Reboot a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.reboot', args)

    def reset(self, uuid):
        """
        Reset (Force reboot) a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.reset', args)

    def pause(self, uuid):
        """
        Pause a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.pause', args)

    def resume(self, uuid):
        """
        Resume a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        self._client.sync('kvm.resume', args)

    def info(self, uuid):
        """
        Get info about a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        return self._client.json('kvm.info', args)

    def infops(self, uuid):
        """
        Get info per second about a kvm domain by uuid
        :param uuid: uuid of the kvm container (same as the used in create)
        :return:
        """
        args = {
            'uuid': uuid,
        }
        self._domain_action_chk.check(args)

        return self._client.json('kvm.infops', args)

    def attach_disk(self, uuid, media):
        """
        Attach a disk to a machine
        :param uuid: uuid of the kvm container (same as the used in create)
        :param media: the media object to attach to the machine
                      media object is a dict of {url, and type} where type can be one of 'disk', or 'cdrom', or empty (default to disk)
                      examples: {'url': 'nbd+unix:///test?socket=/tmp/ndb.socket'}, {'type': 'cdrom': '/somefile.iso'}
        :return:
        """
        args = {
            'uuid': uuid,
            'media': media,
        }
        self._man_disk_action_chk.check(args)

        self._client.sync('kvm.attach_disk', args)

    def detach_disk(self, uuid, media):
        """
        Detach a disk from a machine
        :param uuid: uuid of the kvm container (same as the used in create)
        :param media: the media object to attach to the machine
                      media object is a dict of {url, and type} where type can be one of 'disk', or 'cdrom', or empty (default to disk)
                      examples: {'url': 'nbd+unix:///test?socket=/tmp/ndb.socket'}, {'type': 'cdrom': '/somefile.iso'}
        :return:
        """
        args = {
            'uuid': uuid,
            'media': media,
        }
        self._man_disk_action_chk.check(args)

        self._client.sync('kvm.detach_disk', args)

    def add_nic(self, uuid, type, id=None, hwaddr=None):
        """
        Add a nic to a machine
        :param uuid: uuid of the kvm container (same as the used in create)
        :param type: nic_type # default, bridge, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
         param id: id # depends on the type, bridge name (bridge type) zerotier network id (zertier type), the vlan tag or the vxlan id
         param hwaddr: the hardware address of the nic
        :return:
        """
        args = {
            'uuid': uuid,
            'type': type,
            'id': id,
            'hwaddr': hwaddr,
        }
        self._man_nic_action_chk.check(args)

        return self._client.json('kvm.add_nic', args)

    def remove_nic(self, uuid, type, id=None, hwaddr=None):
        """
        Remove a nic from a machine
        :param uuid: uuid of the kvm container (same as the used in create)
        :param type: nic_type # default, bridge, vlan, or vxlan (note, vlan and vxlan only supported by ovs)
         param id: id # depends on the type, bridge name (bridge type) zerotier network id (zertier type), the vlan tag or the vxlan id
         param hwaddr: the hardware address of the nic
        :return:
        """
        args = {
            'uuid': uuid,
            'type': type,
            'id': id,
            'hwaddr': hwaddr,
        }
        self._man_nic_action_chk.check(args)

        return self._client.json('kvm.remove_nic', args)

    def limit_disk_io(self, uuid, media, totalbytessecset=False, totalbytessec=0, readbytessecset=False, readbytessec=0, writebytessecset=False,
                      writebytessec=0, totaliopssecset=False, totaliopssec=0, readiopssecset=False, readiopssec=0, writeiopssecset=False, writeiopssec=0,
                      totalbytessecmaxset=False, totalbytessecmax=0, readbytessecmaxset=False, readbytessecmax=0, writebytessecmaxset=False, writebytessecmax=0,
                      totaliopssecmaxset=False, totaliopssecmax=0, readiopssecmaxset=False, readiopssecmax=0, writeiopssecmaxset=False, writeiopssecmax=0,
                      totalbytessecmaxlengthset=False, totalbytessecmaxlength=0, readbytessecmaxlengthset=False, readbytessecmaxlength=0,
                      writebytessecmaxlengthset=False, writebytessecmaxlength=0, totaliopssecmaxlengthset=False, totaliopssecmaxlength=0,
                      readiopssecmaxlengthset=False, readiopssecmaxlength=0, writeiopssecmaxlengthset=False, writeiopssecmaxlength=0, sizeiopssecset=False,
                      sizeiopssec=0, groupnameset=False, groupname=''):
        """
        Remove a nic from a machine
        :param uuid: uuid of the kvm container (same as the used in create)
        :param media: the media to limit the diskio
        :return:
        """
        args = {
            'uuid': uuid,
            'media': media,
            'totalbytessecset': totalbytessecset,
            'totalbytessec': totalbytessec,
            'readbytessecset': readbytessecset,
            'readbytessec': readbytessec,
            'writebytessecset': writebytessecset,
            'writebytessec': writebytessec,
            'totaliopssecset': totaliopssecset,
            'totaliopssec': totaliopssec,
            'readiopssecset': readiopssecset,
            'readiopssec': readiopssec,
            'writeiopssecset': writeiopssecset,
            'writeiopssec': writeiopssec,
            'totalbytessecmaxset': totalbytessecmaxset,
            'totalbytessecmax': totalbytessecmax,
            'readbytessecmaxset': readbytessecmaxset,
            'readbytessecmax': readbytessecmax,
            'writebytessecmaxset': writebytessecmaxset,
            'writebytessecmax': writebytessecmax,
            'totaliopssecmaxset': totaliopssecmaxset,
            'totaliopssecmax': totaliopssecmax,
            'readiopssecmaxset': readiopssecmaxset,
            'readiopssecmax': readiopssecmax,
            'writeiopssecmaxset': writeiopssecmaxset,
            'writeiopssecmax': writeiopssecmax,
            'totalbytessecmaxlengthset': totalbytessecmaxlengthset,
            'totalbytessecmaxlength': totalbytessecmaxlength,
            'readbytessecmaxlengthset': readbytessecmaxlengthset,
            'readbytessecmaxlength': readbytessecmaxlength,
            'writebytessecmaxlengthset': writebytessecmaxlengthset,
            'writebytessecmaxlength': writebytessecmaxlength,
            'totaliopssecmaxlengthset': totaliopssecmaxlengthset,
            'totaliopssecmaxlength': totaliopssecmaxlength,
            'readiopssecmaxlengthset': readiopssecmaxlengthset,
            'readiopssecmaxlength': readiopssecmaxlength,
            'writeiopssecmaxlengthset': writeiopssecmaxlengthset,
            'writeiopssecmaxlength': writeiopssecmaxlength,
            'sizeiopssecset': sizeiopssecset,
            'sizeiopssec': sizeiopssec,
            'groupnameset': groupnameset,
            'groupname': groupname,
        }
        self._limit_disk_io_action_chk.check(args)

        self._client.sync('kvm.limit_disk_io', args)

    def migrate(self, uuid, desturi):
        """
        Migrate a vm to another node
        :param uuid: uuid of the kvm container (same as the used in create)
        :param desturi: the uri of the destination node
        :return:
        """
        args = {
            'uuid': uuid,
            'desturi': desturi,
        }
        self._migrate_action_chk.check(args)

        self._client.sync('kvm.migrate', args)

    def list(self):
        """
        List configured domains

        :return:
        """
        return self._client.json('kvm.list', {})


class Logger:
    _level_chk = typchk.Checker({
        'level': typchk.Enum("CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG"),
    })

    def __init__(self, client):
        self._client = client

    def set_level(self, level):
        """
        Set the log level of the g8os
        Note: this level is for messages that ends up on screen or on log file

        :param level: the level to be set can be one of ("CRITICAL", "ERROR", "WARNING", "NOTICE", "INFO", "DEBUG")
        """
        args = {
            'level': level,
        }
        self._level_chk.check(args)

        return self._client.json('logger.set_level', args)

    def reopen(self):
        """
        Reopen log file (rotate)
        """
        return self._client.json('logger.reopen', {})

    def subscribe(self, queue=None):
        """
        Subscribe to the aggregated log stream. On subscribe a ledis queue will be fed with all running processes
        logs. Always use the returned queue name from this method, even if u specified the queue name to use

        Note: it is legal to subscribe to the same queue, but would be a bad logic if two processes are trying to
        read from the same queue.

        :param queue: Your unique queue name, otherwise, a one will get generated for your
        :return: queue name to pull from
        """

        return self._client.json('logger.subscribe', {'queue': queue})

    def unsubscribe(self, queue):
        """
        Unsubscribe will kill the queue on node zero, further reading on that queue will just get what has been
        queued before calling unsubscribe, after that reading on that queue will not return anything.

        :param queue: Queue name as returned from self.subscribe
        :return:
        """
        return self._client.json('logger.unsubscribe', {'queue': queue})



class Nft:
    _port_chk = typchk.Checker({
        'port': int,
        'interface': typchk.Or(str, typchk.IsNone()),
        'subnet': typchk.Or(str, typchk.IsNone()),
    })

    def __init__(self, client):
        self._client = client

    def open_port(self, port, interface=None, subnet=None):
        """
        open port
        :param port: then port number
        :param interface: an optional interface to open the port for
        :param subnet: an optional subnet to open the port for
        """
        args = {
            'port': port,
            'interface': interface,
            'subnet': subnet,
        }
        self._port_chk.check(args)

        return self._client.json('nft.open_port', args)

    def drop_port(self, port, interface=None, subnet=None):
        """
        close an opened port (takes the same parameters passed in open)
        :param port: then port number
        :param interface: an optional interface to close the port for
        :param subnet: an optional subnet to close the port for
        """
        args = {
            'port': port,
            'interface': interface,
            'subnet': subnet,
        }
        self._port_chk.check(args)

        return self._client.json('nft.drop_port', args)

    def list(self):
        """
        List open ports
        """
        return self._client.json('nft.list', {})

    def rule_exists(self, port, interface=None, subnet=None):
        """
        Check if a rule exists (takes the same parameters passed in open)
        :param port: then port number
        :param interface: an optional interface
        :param subnet: an optional subnet
        """
        args = {
            'port': port,
            'interface': interface,
            'subnet': subnet,
        }
        self._port_chk.check(args)

        return self._client.json('nft.rule_exists', args)


class Config:

    def __init__(self, client):
        self._client = client

    def get(self):
        """
        Get the config of g8os
        """
        return self._client.json('config.get', {})


class AggregatorManager:
    _query_chk = typchk.Checker({
        'key': typchk.Or(str, typchk.IsNone()),
        'tags': typchk.Map(str, str),
    })

    def __init__(self, client):
        self._client = client

    def query(self, key=None, **tags):
        """
        Query zero-os aggregator for current state object of monitored metrics.

        Note: ID is returned as part of the key (if set) to avoid conflict with similar metrics that
        has same key. For example, a cpu core nr can be the id associated with 'machine.CPU.percent'
        so we can return all values for all the core numbers in the same dict.

        U can filter on the ID as a tag
        :example:
            self.query(key=key, id=value)

        :param key: metric key (ex: machine.memory.ram.available)
        :param tags: optional tags filter
        :return: dict of {
            'key[/id]': state object
        }
        """
        args = {
            'key': key,
            'tags': tags,
        }
        self._query_chk.check(args)

        return self._client.json('aggregator.query', args)


class Client(BaseClient):
    _raw_chk = typchk.Checker({
        'id': str,
        'command': str,
        'arguments': typchk.Any(),
        'queue': typchk.Or(str, typchk.IsNone()),
        'max_time': typchk.Or(int, typchk.IsNone()),
        'stream': bool,
        'tags': typchk.Or([str], typchk.IsNone()),
    })

    def __init__(self, host, port=6379, password="", db=0, ssl=True, timeout=None, testConnectionAttempts=3):
        super().__init__(timeout=timeout)

        socket_timeout = (timeout + 5) if timeout else 15
        socket_keepalive_options = dict()
        if hasattr(socket, 'TCP_KEEPIDLE'):
            socket_keepalive_options[socket.TCP_KEEPIDLE] = 1
        if hasattr(socket, 'TCP_KEEPINTVL'):
            socket_keepalive_options[socket.TCP_KEEPINTVL] = 1
        if hasattr(socket, 'TCP_KEEPIDLE'):
            socket_keepalive_options[socket.TCP_KEEPIDLE] = 1
        self._redis = redis.Redis(host=host, port=port, password=password, db=db, ssl=ssl,
                                  socket_timeout=socket_timeout,
                                  socket_keepalive=True, socket_keepalive_options=socket_keepalive_options)
        self._container_manager = ContainerManager(self)
        self._bridge_manager = BridgeManager(self)
        self._disk_manager = DiskManager(self)
        self._btrfs_manager = BtrfsManager(self)
        self._zerotier = ZerotierManager(self)
        self._kvm = KvmManager(self)
        self._logger = Logger(self)
        self._nft = Nft(self)
        self._config = Config(self)
        self._aggregator = AggregatorManager(self)

        if testConnectionAttempts:
            for _ in range(testConnectionAttempts):
                try:
                    self.ping()
                except:
                    pass
                else:
                    return
            raise ConnectionError("Could not connect to remote host %s" % host)

    @property
    def container(self):
        """
        Container manager
        :return:
        """
        return self._container_manager

    @property
    def bridge(self):
        """
        Bridge manager
        :return:
        """
        return self._bridge_manager

    @property
    def disk(self):
        """
        Disk manager
        :return:
        """
        return self._disk_manager

    @property
    def btrfs(self):
        """
        Btrfs manager
        :return:
        """
        return self._btrfs_manager

    @property
    def zerotier(self):
        """
        Zerotier manager
        :return:
        """
        return self._zerotier

    @property
    def kvm(self):
        """
        KVM manager
        :return:
        """
        return self._kvm

    @property
    def logger(self):
        """
        Logger manager
        :return:
        """
        return self._logger

    @property
    def nft(self):
        """
        NFT manager
        :return:
        """
        return self._nft

    @property
    def config(self):
        """
        Config manager
        :return:
        """
        return self._config

    @property
    def aggregator(self):
        """
        Aggregator manager
        :return:
        """
        return self._aggregator

    def raw(self, command, arguments, queue=None, max_time=None, stream=False, tags=None, id=None):
        """
        Implements the low level command call, this needs to build the command structure
        and push it on the correct queue.

        :param command: Command name to execute supported by the node (ex: core.system, info.cpu, etc...)
                        check documentation for list of built in commands
        :param arguments: A dict of required command arguments depends on the command name.
        :param queue: command queue (commands on the same queue are executed sequentially)
        :param max_time: kill job server side if it exceeded this amount of seconds
        :param stream: If True, process stdout and stderr are pushed to a special queue (stream:<id>) so
            client can stream output
        :param tags: job tags
        :param id: job id. Generated if not supplied
        :return: Response object
        """
        if not id:
            id = str(uuid.uuid4())

        payload = {
            'id': id,
            'command': command,
            'arguments': arguments,
            'queue': queue,
            'max_time': max_time,
            'stream': stream,
            'tags': tags,
        }

        self._raw_chk.check(payload)
        flag = 'result:{}:flag'.format(id)
        self._redis.rpush('core:default', json.dumps(payload))
        if self._redis.brpoplpush(flag, flag, DefaultTimeout) is None:
            TimeoutError('failed to queue job {}'.format(id))
        logger.debug('%s >> g8core.%s(%s)', id, command, ', '.join(("%s=%s" % (k, v) for k, v in arguments.items())))

        return Response(self, id)

    def response_for(self, id):
        return Response(self, id)
