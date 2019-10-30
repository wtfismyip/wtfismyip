#!/bin/bash
/usr/lib/unbound/package-helper chroot_setup
/usr/lib/unbound/package-helper root_trust_anchor_update
/usr/sbin/unbound -d
