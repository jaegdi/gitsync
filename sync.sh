#!/usr/bin/env bash

./build/gitsync -base "/tmp/bitbucket-repo-sync" -username jaegdi -passwordfile <(kwalletcli -f admin -e sfbkpassword 2>/dev/null)