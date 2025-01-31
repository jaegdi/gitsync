#!/usr/bin/env bash

# an example call of gitsync to sync all projects and repos in 'repos.txt' and get the password for bitbucket login from kwalletcli

./build/gitsync -base "/tmp/bitbucket-repo-sync" -file repos.txt -username jaegdi -passwordfile <(kwalletcli -f admin -e sfbkpassword 2>/dev/null)