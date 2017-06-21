# ZeroTier Commands

Available commands:

- [zerotier.info](#info)
- [zerotier.join](#join)
- [zerotier.leave](#leave)
- [zerotier.list](#list)


<a id="info"></a>
## zerotier.info

Information about the ZeroTier network.

<a id="join"></a>
## zerotier.join

Joins a given ZeroTier network.

Arguments:
```javascript
{
	"network": "{network}",
}
```


Values:
- **network**: ZeroTier network ID


<a id="leave"></a>
## zerotier.leave

Leaves a given ZeroTier network.

Arguments:
```javascript
{
	"network": "{network}",
}
```


Values:
- **network**: ZeroTier network ID

<a id="list"></a>
## zerotier.list

Lists all joined ZeroTier networks.
