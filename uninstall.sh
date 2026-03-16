#!/bin/sh
set -e

if [ ! -f /usr/local/bin/lighthouse ]; then
    echo "Lighthouse is not installed."
    exit 0
fi

sudo rm /usr/local/bin/lighthouse
echo "Lighthouse uninstalled."