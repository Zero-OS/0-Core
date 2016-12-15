#!/bin/sh
set -e

#PATH=/tmp/ZeroTierOne/:$PATH
nsname=$(echo $1 | tr " " "-")
zeronet="$2"
zerohome="/tmp/zerotier-home-$nsname"

#
# sanity check
#
if [ -z "$nsname" ]; then
    echo "[-] missing namespace name"
    exit 1
fi

#
# initializing
#
echo "[+] initializing daemon for namespace: $nsname ($zerohome)"
zerotier-one -p0 "$zerohome" 2> /dev/null &
daemonpid=$!


function cleanup {
	echo "[+] cleaning up"
	kill $daemonpid
	rm -rf "$zerohome"
	exit 0
}

trap cleanup SIGTERM

#
# waiting for daemon to be ready
#
while ! zerotier-cli -D$zerohome listnetworks 2>&1 | grep ^200 > /dev/null; do
    sleep 0.2
done

echo "[+] daemon initialized: pid $daemonpid"

#
# joining network
#
echo "[+] requesting address for network: $zeronet"
req=$(zerotier-cli -D$zerohome join $zeronet)

if [ -z "$(echo $req | grep ^200)" ]; then
    echo "[-] cannot join network: $req"
fi

echo "[+] waiting connectivity"
while ! zerotier-cli -D$zerohome listnetworks | grep 'OK' | grep $zeronet > /dev/null; do
    sleep 0.5
done

data=$(zerotier-cli -D$zerohome listnetworks | grep $zeronet | tail -1)
zeroiface=$(echo $data | awk '{ print $8 }')
zeroaddr=$(echo $data | awk '{ print $9 }' | cut -d, -f 2)

echo "[+] network connected: $zeroaddr via $zeroiface"

#
# namespace isolation
#
echo "[+] creating namespace"
ip l set dev $zeroiface netns $nsname
ip netns exec $nsname ip l set dev $zeroiface up
ip netns exec $nsname ip a add $zeroaddr dev $zeroiface

wait
cleanup
