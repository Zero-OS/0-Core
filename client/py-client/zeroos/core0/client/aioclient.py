import ssl
import json
import aioredis
import logging
import sys

from . import typchk
from .client import BaseClient

logger = logging.getLogger('g8core')


class AioClient(BaseClient):
    _raw_chk = typchk.Checker({
        'id': str,
        'command': str,
        'arguments': typchk.Any(),
        'queue': typchk.Or(str, typchk.IsNone()),
        'max_time': typchk.Or(int, typchk.IsNone()),
        'stream': bool,
        'tags': typchk.Or([str], typchk.IsNone()),
    })

    def __init__(self, loop, host, port=6379, password="", db=0, ctx=None, timeout=None, testConnectionAttempts=3):
        super().__init__(timeout=timeout)

        socket_timeout = (timeout + 5) if timeout else 15

        self.testConnectionAttempts = testConnectionAttempts

        self._redis = None
        self.host = host
        self.port = port
        self.password = password
        self.db = db
        if ctx is None:
            ctx = ssl.create_default_context()
            ctx.check_hostname = False
            ctx.verify_mode = ssl.CERT_NONE
        self.ssl = ctx
        self.timeout = socket_timeout
        self.loop = loop

    async def get(self):
        if self._redis is not None:
            return self.redis

        self._redis = await aioredis.create_redis((self.host, self.port),
                                                  loop=self.loop,
                                                  password=self.password,
                                                  db=self.db,
                                                  ssl=self.ssl,
                                                  timeout=self.timeout)
        return self._redis

    async def global_stream(self, callback):
        """
        Runtime copy of node logging messages.
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
        :return: None
        """
        def default_callback(level, line, meta):
            w = sys.stdout if level == 1 else sys.stderr
            w.write(line)
            w.write('\n')

        if callback is None:
            callback = default_callback

        if not callable(callback):
            raise Exception('callback must be callable')

        self._redis = await self.get()
        queue = "core:logs"

        while True:
            data = await self._redis.blpop(queue, 10)
            a, body = data
            payload = json.loads(body.decode())
            message = payload['message']
            line = message['message']
            meta = message['meta']
            callback(meta >> 16, line, meta & 0xff)
