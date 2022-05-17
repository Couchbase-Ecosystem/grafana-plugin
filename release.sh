#!/usr/bin/env bash

######################################################################
# @author      : chedim (chedim@couchbaser)
# @file        : release
# @created     : Tuesday May 17, 2022 10:00:57 EDT
#
# @description : Packages the plugin for a release
######################################################################


tmp=$(mktemp -d)

mkdir "$tmp/couchbase-datasource"

pushd couchbase-datasource
yarn build
mage -v

cp -r dist "$tmp/couchbase-datasource/"
popd

zip couchbase-datasource.zip "$tmp/couchbase-datasource" -r
