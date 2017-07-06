# Join the ZeroTier Management Network

In order to interact with a Zero-OS node your client machine needs to join ZeroTier management network of the node.

See the [ZeroTier website](https://zerotier.com/) for installation instructions.

Once installed check status of the ZeroTier daemon using `zerotier-cli`, the ZeroTier command line tool:
```shell
zerotier-cli info
```

In case the ZeroTier daemon is not yet running, launch it:
```shell
zerotier-one -d
```

Check if your client machine already joined a ZeroTier network:
```shell
zerotier-cli listnetworks
```

If no ZeroTier network was yet joined, join the ZeroTier management network identified with the ZeroTier network ID:
```shell
export ZEROTIER_NETWORK_ID="..."
zerotier-cli join $ZEROTIER_NETWORK_ID
```

You will now need to go to `https://my.zerotier.com/network/$ZEROTIER_NETWORK_ID` in order to authorize the join request.
