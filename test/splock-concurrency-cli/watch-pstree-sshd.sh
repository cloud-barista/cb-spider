#!/bin/bash

watch "pstree `pgrep -f "/usr/sbin/sshd -D"`"
