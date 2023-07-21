#!/bin/sh

#
# NOTE: use this script to import the user `tenant_admin` and its group `tenant_admin_group` to make the full
# example work out-of-the-box. If the IDs don't match the IDs of your installation, please check them using your browser
#

terraform import axual_user.tenant_admin 5370bc1c9d8347c4b7169fc6f82906ee
terraform import axual_group.tenant_admin_group dd84b3ee8e4341fbb58704b18c10ec5c