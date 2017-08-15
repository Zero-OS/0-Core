# Backup
Containers can do a backup to a restic repository by calling the backup method with a 
valid `restic` url.

Backing up containers takes a full snapshot of the container files so it can be fully restored
without the flist or the ardb storage. This will allow restoring the backup in remote env
where the flist or the storage are not in sync with the source env.

Backup and restore of containers uses a slightly modified version of the `restic` repos defined
 in their [manual](https://restic.readthedocs.io/en/latest/manual.html). 

## Example 
```python
job = cl.container.backup(
    container_id,
    'file:///path/to/restic/repo?password=<password>',
)

snapshot = job.get()
```

This will return the snapshot ID (to be used later for restore)

# Restore
## Full restore of containers
To fully restore the container you need to simply call the `restore` method with a valid restic repo URL.
The restic URLs are a slightly modified version of the repos urls defined in the restic manual.

A full restore will create an exact copy of the backed-up container (same data and config). Basically
it's a full restore of creation flags plus the container data (network setup, ports, privileges, etc...)

> Note: This does NOT mean container will get the same IPs, it's as if it's created with the same network setup
so for default network, or any network that uses dhcp, a new IPs will get assigned unless a static IP was originally
configure.

## Data only restore of containers
To create a new container from the backup but with new configuration, you simply need to call create normally
but with the `root_url` to point to a restic backup URL instead of an flist.

> IMPORTANT: to distinguish between an flist and a restic repo, all restic repos must be prefixed
with `restic:`

>Example: `restic:file:///path/to/restic?password=pass#snapshot-id`

# Restic URL
On calling backup we (theoretically) support all repo types that are supported by `restic` with a slight change

for example, to backup to a local repository a path should be prefixed with `file://` and the password
is passed as a url param 

- `file:///path/to/restic?password=<password>`

On restore we use the same url formatting plus a url fragment for the snapshot ID

- `files:///path/to/restic?password=<password>#snapshot-id`

# Full example with sftp
First of all make sure that the zero-os node can ssh to the remote host where the restic repo
exits.

- On zero-os node, if the repo was not yet initialized do the following
```bash
restic -r user@host:/tmp/backup init
```
- Enter and keep the repo password because we will need it in backup and restore.
- Using the client create a container (flist and config are totally up to you).

## taking a backup
```python

job = cl.contrainer.backup(
    container_id,
    'sftp:user@host:/tmp/backup?password=<password>')

snapshot = job.get()
```
> Depends on how big your container is and speed of your upload link to remote backup host
the `job.get()` can timeout, try to call `job.get()` again or use a big timeout.

## Data only restore
```python
url = 'restic:sftp:user@host:/tmp/backup?password=<password>#%s' % snapshot
cl.container.create(url) # pass other create config according to your needs
```

## Full restore
```python
url = 'sftp:user@host:/tmp/backup?password=<password>#%s' % snapshot
cl.container.restore(url)
```

> IMPORTANT: Notice how the `url` is different in case of data only or full restore. That is because
the container.create() must be able to tell an flist and restic urls apart.
