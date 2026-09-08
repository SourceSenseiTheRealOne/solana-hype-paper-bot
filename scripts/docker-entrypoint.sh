#!/bin/sh
set -eu

install -d -m 0750 -o app -g app /app/var/log /app/var/reports
exec su-exec app:app /app/paper-bot
