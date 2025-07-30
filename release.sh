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


if  [ ! -f plugins/couchbase-datasource/versions/${CBVER}/download ]; then
  mkdir "$tmp/couchbase-datasource"

  pushd couchbase-datasource
  yarn build
  mage -v


  rm ../couchbase-datasource.zip
  zip ../couchbase-datasource.zip dist -r
  popd
  MD5=$(md5 -q couchbase-datasource.zip)

  mkdir -p plugins/couchbase-datasource/versions/${CBVER}
  cp couchbase-datasource.zip plugins/couchbase-datasource/versions/${CBVER}/download

  cat plugins/repo/index.html | jq ".plugins[0].versions+=[{\"version\":\"${CBVER}\",\"download\":{\"any\": {\"url\": \"https://github.com/Couchbase-Ecosystem/grafana-plugin/raw/refs/heads/main/plugins/couchbase-datasource/versions/${CBVER}/download\", \"md5\":\"${MD5}\"}}}]" > "$tmp/cb-repo-index.json"
  mv "$tmp/cb-repo-index.json" plugins/repo/index.html

  cat plugins/repo/couchbase-datasource/index.html | jq ".versions+=[{\"arch\":{\"any\":{}},\"version\":\"${CBVER}\"}]" > "$tmp/cbds-versions.json"
  mv "$tmp/cbds-versions.json" plugins/repo/couchbase-datasource/index.html

  cat plugins/couchbase-datasource/versions/index.html | jq ".items+=[{\"packages\": {\"any\": {\"downloadUrl\": \"https://github.com/couchbase-ecosystem/grafana-plugin/raw/refs/heads/main/plugins/couchbase-datasource/versions/${CBVER}/download\", \"md5\": \"${MD5}\"}},\"version\": \"${CBVER}\"}]" > $tmp/versions.json
  mv $tmp/versions.json plugins/couchbase-datasource/versions/index.html

  git add plugins
fi

rm -Rf $tmp
rm couchbase-datasource.zip
