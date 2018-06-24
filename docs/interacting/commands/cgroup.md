# Cgroup Commands

Available commands:

- Commands
    - [ensure](#ensure)
    - [list](#list)
    - [remove](#remove)
    - [tasks](#tasks)
    - [task-add](#task-add)
    - [task-remove](#task-remove)
    - [reset](#reset)
    - [memory](#memory)
    - [cpuset](#cpuset)
- Examples
    - [Memory CGroup](#memory-cgroup)
    - [Cpuset CGroup](#cpuset-cgroup)
    - [Containers CGroups](#containers-cgroups)

## ensure
Make sure that a cgroup exists, and create it if it does not. The ensure method does not change configuration of the group if it exists.

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup

## list

Lists all available cgroups on a host. It takes no arguments.

## remove

Removes a cgroup. Note that any process that is attached to this cgroups are moved to the default cgroup which has no limitation

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup

## tasks

List all tasks/processes that are added to this cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup


### task-add
Add a new task (process ID) to a cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
    'pid': {pid},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup
- **{pid}**: PID to add


### task-remove
Remove a task/process (process ID) from a cgroup

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
    'pid': {pid},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup
- **{pid}**: PID to remove

### reset
Resets all limitations on the cgroup to the default values

Arguments:
```javascript
{
    'subsystem': {subsystem},
    'name': {name},
}
```

Values:
- **{subsystem}**: the Cgroup subsystem currently only support (`memory`, and `cpuset`)
- **{name}**: name of the cgroup


### memory
Get/Set memory limits on a memory cgroup. A call to this method without a `mem` value will not change the current limitations.
A call to this method will always return the current values.

Arguments:
```javascript
{
    'name': {name},
    'mem': {mem},
    'swap': {swap},
}
```

Values:
- **{name}**: name of a memory cgroup
- **{mem}**: Set memory limit to the given value (in bytes), ignore if 0
- **{swap}**: Set swap limit to the given value (in bytes) (only if mem is not zero)


### cpuset
Get cpuset cgroup specification/limitation the call to this method will always GET the current set values for both cpus and mems If cpus, or mems is NOT NONE value it will be set as the spec for that attribute

Arguments:
```javascript
{
    'name': {name},
    'cpus': {cpus},
    'mems': {mems},
}
```

Values:
- **{name}**: name of a cpuset cgroup
- **{cpus}**: Set cpus affinity limit to the given value (0, 1, 0-10, etc...)
- **{mems}**: Set mems affinity limit to the given value (0, 1, 0-10, etc...)

# Examples
In general the process of controlling/limiting a process resources goes as follows:
- Create a cgroup of proper type (only memory, or cpuset are supported so far)
- Configure the cgroup (based on the type)
- Add PID to a group

## Memory CGroup
A memory cgroup can be used to set a limit on a process memory usage. By simply creating a cgroup, and set it's limit and then assign the PID you want to control to that cgroup.

```python
cl.cgroup.ensure('memory', '100m') # create a memory cgroup with name '100m'
cl.cgroup.memory('100m', 100*1024*1024) # set the limit of physical memory to 100 MB, and 0 MB swap space
cl.cgroup.task_add('memory', '100m', pid) # add process to cgroup
```

> Notice that all actions that is generic regardless of the cgroup subsystem type, requires a `subsystem` argument. While calls that is very specific to a certain types (like the `memory` method above) only requires the name.

## CPUSet CGroup
A cpuset cgroup can be used to set the process `affinity` to a specific physical cpus and mems (memory nodes). The cpuset cgroup has more flags to configure in real life, but we only expose the cpus and mems nodes for now, until we have a use case for the other flags.

```python
cl.cgroup.ensure('cpuset', 'cpu0') # create a cpuset cgroup of name 'cpu0'
cl.cgroup.reset('cpuset', 'cpu0') # reset the cgroup to default values (all cpus, all mems)
cl.cgroup.cpuset('cpu0', '0') # set the cpu affinity to cpu 0
cl.cgroup.task_add('cpuset', 'cpu0', pid) # add process to cgroup
```

## Containers CGroups
To add a container to a cgroup(s), you need to first configure the required cgroups (memory and/or cpuset) then use the [container create API](container.md#create) to pass the cgroups you want to join. It's possible to add the container coreX process PID to a cgroup using the `task_add` method, but this will only affect the coreX process and it's future children, any processes that have been created already by the coreX process will not get affected. Hence it's much better to set the cgroups via the container.create API which grantees that ALL processes in the container will be part of the configured cgroups.