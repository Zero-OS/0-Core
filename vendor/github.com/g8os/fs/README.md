![Build Status](https://travis-ci.org/g8os/fs.svg?branch=master)

# aysfs
Caching filesystem which will be used by ays, mainly used to deploy applications in a grid

# How to
config file example
```
[[mount]]
     path="/opt"
     flist="/root/jumpscale__base.flist"
     backend="main"
     #stor="stor1"
     mode = "OL"
     trim_base = true

[backend.main]
    path="/tmp/aysfs_main"
    stor="stor1"
    #namespace="testing"
    namespace="dedupe"
    
    upload=true
    encrypted=false
    # encrypted=true
    user_rsa="user.rsa"
    store_rsa="store.rsa"

    aydostor_push_cron="@every 1m"
    cleanup_cron="@every 1m"
    cleanup_older_than=1 #in hours

[aydostor.stor1]
    addr="http://192.168.122.1:8080/"
    #addr="http://192.168.0.182:8080/"
    login="zaibon"
    passwd="supersecret"
```
## Stores 
Stores defines the places where files can be retrieved. A store is defined with an `aydostor` section as following
```toml
[aydostor.storX]
   addr="http://stor.host/"
   login=""
   passwd=""
```
A single store can be used by multiple backend using the store name

## Backends
A backend defines the local files cache. It defines how to retrieve the files from the stores, and which store to use. also defined how to push changes back to the store and if files should be pushed back to the store in the first place.

### Fuse lib
There are two fuse lib we use, https://bazil.org/fuse/ and https://github.com/hanwen/go-fuse (default).
To use bazil's lib, we need to specify `lib="bazil"` in the config.
example:
```
[backend.main]
    path="/root/aysfs_main"
    stor="stor1"
    namespace="js8_opt"

    lib="bazil"
```

## Mounts
A list of mount points, each mount defines what backend to use and mount `mode`. example:
```toml
[[mount]]
     path="/opt"
     flist="/root/jumpscale__base.flist"
     backend="main"
     #stor="stor1"
     mode = "OL"
```
*Flist* is required in case `acl=RO` or `acl=OL`
*mode* can be one of `RW` (ReadWrite), `RO` (ReadOnly) or, `OL` (Overlay)

## Starting fuse layer
```./aysfs -config config.toml ```

###To enable pprof tool, add the -pprof flag to the command  
```./aysfs -pprof /opt```  
and go to http://localhost:6060/debug/pprof
