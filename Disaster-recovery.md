# PANIC

say you fucked up the fileserver, follow the following steps to restore

## Initial startup

### disabling overlayfs

after reinstallation start by disabling overlayFS

```bash
sudo 'echo "overlayfs=disable" > /.init_wipedata'
sudo reboot
```

### installing dependencies

after the system comes back online execute the following steps to install the necessary dependencies

```bash
sudo apt update
sudo apt upgrade -y
echo "deb [signed-by=/usr/share/keyrings/azlux-archive-keyring.gpg] http://packages.azlux.fr/debian/ bookworm main" | sudo tee /etc/apt/sources.list.d/azlux.list
sudo wget -O /usr/share/keyrings/azlux-archive-keyring.gpg  https://azlux.fr/repo.gpg

sudo apt update
sudo apt install nfs-kernel-server rpcbind mdadm log2ram nano
```

### bringing up the softraid array

now lets bring back up our softraid array, this isn´t ideal, but with lack of ZFS support back in the day it was the best we had

```bash
sudo su
mdadm --assemble --scan
mdadm --detail --scan >> /etc/mdadm/mdadm.conf
mkdir /mnt/storage
echo "UUID=0f534007-59d8-42ee-b10e-c142cb46611a       /mnt/storage    btrfs   defaults        0       0" >> /etc/fstab
mount -a
```

At this point it's probably a good idea to check all the files are still there. in /mnt/storage.

### setting the default ip

The way the network is currently setup it's not segmented well.
This is a todo for the next iteration.

```bash
sudo nmcli connection modify "Wired connection 1" \
ipv4.method "manual" \
ipv4.addresses "192.168.0.7/16" \
ipv4.gateway "192.168.0.1" \
ipv4.dns "1.1.1.1,1.0.0.1"

sudo systemctl restart NetworkManager
reboot
```

## NFS

right now, we use nfs instead of something fancy like ceph.
this means we have to do some setup here

```bash
perl -pi -e 's/^OPTIONS/#OPTIONS/' /etc/default/rpcbind

echo "/exports *(ro,no_root_squash,no_subtree_check,fsid=0)" >> /etc/exports
sudo mount -o bind /mnt/storage/Media /exports/Media

echo "/exports/Media 192.168.0.0/16(rw,no_root_squash,no_subtree_check) 10.69.0.0/16(rw,no_root_squash,no_subtree_check)" >> /etc/exports

sudo mount -o bind /mnt/storage/config /exports/config
echo "/exports/config 192.168.0.0/16(rw,no_root_squash,no_subtree_check) 10.69.0.0/16(rw,no_root_squash,no_subtree_check)" >> /etc/exports
exportfs -a
reboot
```

This exports the required directories from the host.

With this being done you should be able to follow the rest of [Getting started](./Getting-started.md)