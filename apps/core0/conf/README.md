You can place here partial toml files that will get dynamically loaded by the agent
Those partial toml files can only define new extensions and startup commands (similar to what can be defined in the main agent config)
that the agent will pickup automatically and register.

It's an easy way to hook scheduled tasks.

## Example
write this section in any file (suffixed .toml) under this folder

```toml
[startup.hearbeat]
name = "execute"
[startup.hearbeat.args]
    name = "touch"
    args = ["/tmp/started"]
    recurring_period = 10

[extensions.hi]
binary = "echo"
args = ["20::\"Hi!\""]
```

This will do 2 things
1- Startup a periodic task that does an `execute touch /tmp/started` which is scheduled to run every 10 seconds
2- It registers an extension `hi` that when called will return `"Hi!"`

To test the extension does you can do something like

```python
cmd = client.cmd(gid, nid, 'hi', acclient.RunArgs())
job = cmd.get_next_result()
print job.data
#should output "Hi!"
```
