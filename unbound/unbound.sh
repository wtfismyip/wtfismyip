#!/bin/bash
/usr/lib/unbound/package-helper chroot_setup
/usr/lib/unbound/package-helper root_trust_anchor_update
CPUS=`lscpu | grep ^CPU\(s | cut -d : -f2 | sed "s/^ *//g"`
sed -i "s/CPUS/$CPUS/g" /etc/unbound/unbound.conf
/usr/sbin/unbound -d
