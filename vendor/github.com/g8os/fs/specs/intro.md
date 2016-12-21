# G8OS Filesystem
The basic idea of the file system is as follows
- When the filesystem is mounted, a `flist` file is loaded to build the filesystem tree using the provided information in the flist
- Each row in the flist is formated as `<path>,<hash>,<size in bytes>`
- We exapand this flist into metadata files in the backend cache for fast access.
- Each metadata file will have the flist entry data in a toml format.
- Metadata files is used to track modifications to the file
- Metadata files has mode `0400` when first created.
- Actual files has mode `0755` by default, and can't be changed (same as directories)

# G8OS modes
Mounts can run in 3 different modes

1- Readonly (RO)
2- Readwrite (RW)
3- Overlay (OL)

## Readonly mode
In readonly mode, a user can't change the content of the file or create a new file. That's the simplest mode. Metadata is never touched or changed. And is only accessed if the actual file doesn't exist.
When a file is accesses:
- If the file exists, all read operations are redirected to the actual file
- If file does not exist, the meta is used to build the download url (using the hash) and file is downloade.
Also in RO mode (and all other modes) a cleaner process starts that vacumes the backend by deleteing actuall files that hasn't been accessed for a long time (1day)

## Readwrite mode
In `RW` mode, the mount starts with an empty mount.
In `RW` mode, the fuse layer starts a `watcher` routine that runs every configurable amount of minutes which does the following
- When awake, it loops over all tracked files that are ready to be uploaded (has been closed or has been open and not modified for a long time (configurable)
- For each ready file, it process it
- Encrypt it
- Compress
- Upload to store
- Create a file metadata with correct hash and size

For newly created files _and_ modified files.
- When a file is modified or created, the meta data file gets a `w` flag to mark the file as modified. This will prevent the filesystem reboot from overriding your modified meta file. and will force the cleaner to skip cleaning up your modified version of the actual file.
- When a file or a directory is deleted. the meta file of the deleted node gets a `x` flag to mark file/dir deletion for the same reason. This will also force the directory listing to not show the deleted file/directory.

> Note: File delete, move, or rename should work even if the actual file is not downloaded from the stor. File is only downloaded in case of `Open`. Other operations can be performed directly on the meta file without the need to download the file. So a file delete makes sure the file meta is marked as deleted, even if the actual file delete failed (because it might not exist at all)

## Overlay mode.
Overlay mode works exactly as the `RW` mode without the uploader worker. So if a new file is created/modifed, it's never uploaded to the store but will always served from local cache. Same meta data roles applies.
