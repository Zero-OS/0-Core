set -e

LABEL="zos-cache"
CACHE="/var/cache"
STORAGEPOOL="/mnt/storagepool"

MNT="${STORAGEPOOL}/${LABEL}"

function error {
    echo $@ >&2
}

function labelmount {
    mount /dev/disk/by-label/$1 $2 > /dev/null 2>&1
    return $?
}

function preparedisk {
    DISK=""
    for disk in `lsblk -pdn -o NAME,ROTA|sort -nk 2|cut -d " " -f 1`; do
        if ! lsblk -n -o TYPE ${disk} | grep part > /dev/null; then
            # disk does not have any parition
            DISK=$disk
            break
        fi
    done

    if [ "$DISK" == "" ]; then
        error "no free disks found"
        exit 1
    fi

    parted -s ${DISK} mktable gpt
    parted -s ${DISK} mkpart primary btrfs 1 100%
    mkfs.btrfs ${DISK}1 -f -L ${LABEL}

    return 0
}

function hook {
    # create required subvols and mount them if not exits

    # 1 - cache subvol
    btrfs subvol create $1/cache || true
    mount $1/cache /var/cache/

    logs=$1/logs
    btrfs subvol create ${logs} || true
    current="log-$(date +%Y%m%d-%H%M)"
    btrfs subvol create ${logs}/${current}
    cp -a /var/log/* ${logs}/${current}/
    mount ${logs}/${current} /var/log
    kill -USR1 1 #log rotation
    return 0
}

function main {
    mkdir -p ${MNT}
    if mountpoint -q ${MNT}; then
        error "${MNT} is already mounted"
        exit 1
    fi

    if ! labelmount ${LABEL} ${MNT}; then
        # no parition found with that label
        # prepare the first availabel disk
        preparedisk
        labelmount ${LABEL} ${MNT}
    fi

    hook $MNT
}

main