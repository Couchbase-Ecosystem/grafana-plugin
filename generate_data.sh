#!/usr/bin/env bash

######################################################################
# @author      : chedim (chedim@couchbaser)
# @file        : generate_data
# @created     : Tuesday Mar 29, 2022 21:11:14 EDT
#
# @description : A time series data generator for Couchbase
######################################################################

cbq=/opt/couchbase/bin/cbq

insert_row() {
  row=$(cat <<EOR
{
  "time": CLOCK_UTC(),
  "count": $(( $RANDOM % 100 ))
}
EOR
)
  echo $row
  query="INSERT INTO ${target?} (KEY, VALUE) VALUES (UUID(), $row);"
  echo "QUERY: $query"
  $cbq -e ${cluster?} -u ${user?} -p ${password?} --no-ssl-verify --script="$query"
}



while(true); do
  insert_row
  sleep ${interval-1}
done
