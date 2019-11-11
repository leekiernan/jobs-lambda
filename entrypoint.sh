#!/bin/sh
set -e

crontab /etc/cron.d/runner-cron && cron -f -L /dev/stdout
