# Streaming Process Output from Zero-OS

The command structure has a `stream` flag. When set to true Zero-OS will push (RPUSH) the command output and error stream to a special queue: `stream:<id>`.

Each entry in the queue is a JSON serialized object:

```javascript
{
	core: 'core-id', //0 for host, and >=0 for a container
	command: 'command-id',
	message: {
		message: 'string', //the log message itself
		epoch: timestamp, //in nanosecond
		meta: uint, //meta flags
	}
}
```

The `meta` attribute is an unsigned 32-bit integer formatted as follows:
- 2 higher order bytes contain the log level (1 for stdout, and 2 for stderr)
- 2 lower order bytes contain flags associated with the message:
	- flag: 0x2 EOF and process has exited with success
	- flag: 0x4 EOF and process has exited with error

Example:
```python
level = meta >> 16
eof = meta & 0x0006 != 0 #EOF regardless the exit state

if eof:
    return

if level == 1:
   stdout.write(message)
elif level == 2:
   stderr.write(message)
```

## Python client streaming

The Python client exposes the stream functionality. Although the stream flag can work with any command (even the internal commands that don't start an external process), the Python client only exposes the `stream` flag on `system` and `bash` methods.

When you start a process via `system` or `bash` the stream method is available on the `Respose` object. The `stream` method is what will start the actual copy of process output from Zero-OS. Calling stream method without having the flag actually set on the call will cause the stream method to block until the process exits, but it will not copy any data since Zero-OS did not actually prepare any data for streaming.

Check docstring of the `stream` method for more details on how to use it.

Example:
```python
response = cl.bash('for i in $(seq 0 10); do echo "ping"; sleep 2s; done', stream=True)
response.stream() # by default will copy stream to sys.stdout and sys.stderr
## output
ping
ping
ping
ping
ping
ping
ping
ping
ping
ping
ping
```
