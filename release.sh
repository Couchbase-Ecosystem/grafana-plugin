#!/usr/bin/env bash

######################################################################
# @author      : chedim (chedim@couchbaser)
# @file        : release
# @created     : Tuesday May 17, 2022 10:00:57 EDT
#
# @description : Packages the plugin for a release
######################################################################


tmp=$(mktemp -d)
CBVER=$(cat couchbase-datasource/package.json | jq -r .version)

mkdir "$tmp/couchbase-datasource"

pushd couchbase-datasource
yarn build
mage -v

cp -r dist "$tmp/couchbase-datasource/"
popd

zip couchbase-datasource.zip "$tmp/couchbase-datasource" -r

mkdir -p plugins/couchbase-datasource/versions/${CBVER}
cp couchbase-datasource.zip plugins/couchbase-datasource/versions/${CBVER}

cat plugins/repo/couchbase-datasource/index.html | jq ".versions+=[{\"arch\":{\"any\":{}},\"version\":\"${CBVER}\"}]" > "$tmp/cbds-versions.json"
mv "$tmp/cbds-versions.json" plugins/repo/couchbase-datasource/index.html