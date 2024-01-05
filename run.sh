#!/usr/bin/env bash

######################################################################
# @author      : chedim (chedim@couchbaser)
# @file        : run
# @created     : Friday Mar 25, 2022 11:41:40 EDT
#
# @description : Starts grafana with the plugin
######################################################################

pushd couchbase-datasource
set -e 
trap popd exit
yarn upgrade || exit 1
yarn install || exit 1
yarn build || exit 1
mage -v || exit 1
popd
trap '' exit
docker compose down
docker compose up 
