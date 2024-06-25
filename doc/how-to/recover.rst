.. _howto-recover:

Each MicroCloud service uses the `dqlite <https://dqlite.io/>` distributed
database for highly-available storage. While the cluster recovery process is
similar for each service, this document only covers cluster recovery for the
`microcloudd` daemon. For cluster recovery procedures for LXD, MicroCeph and
MicroOVN, see:

- `LXD Cluster Recovery <https://documentation.ubuntu.com/lxd/en/latest/howto/cluster_recover/>`
- TBD

If a MicroCloud cluster loses more than a quorum of its database members, then
database operations will no longer be possible on the entire cluster. If the
loss of quorum is temporary (e.g. some members temporarily lose power), database
operations will be restored when the down members come back online.

This document describes how to recover database access if the down members have
been lost without the possibility of recovery (e.g. disk failure).

Recovery Procedure
------------------

1. In order to perform cluster recovery, **all cluster members must be shut down**:

       sudo snap stop microcloud

1. Once all cluster members are shut down, determine which dqlite database is
most up to date. Look for files in `/var/snap/microcloud/common/database` whose
filenames are two numbers separated by a dash (i.e.
`0000000000056436-0000000000056501`). The largest second number in the directory
is the end index of the most recently closed segment (56501 in the example).
Perform the next step on the cluster member with the highest end index.

1. Use the following command **on one cluster member** to reconfigure the dqlite
roles for each member:

       sudo microcloud cluster edit

1. As indicated by the output of the above command, copy
`/var/snap/microcloud/common/recovery_db.tar.gz` to the same path on each
cluster member.

1. Restart microcloud. The recovered database tarball will be loaded on daemon
startup. Once a quorum of voters have been started, the microcloud database
will become available.

       sudo snap start microcloud

Backups
-------
MicroCloud creates a backup of the database directory before performing the
recovery operation to ensure that no data is lost. The backup tarball is created
in `/var/snap/microcloud/common/`. In case the cluster recovery operation fails,
use the following commands to restore the database:

       cd /var/snap/microcloud/common
       sudo mv database broken_db
       sudo tar -xf db_backup.TIMESTAMP.tar.gz
