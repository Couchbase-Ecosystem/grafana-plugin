# Couchbase Grafana Data Source Backend Plugin

[![Build](https://github.com/grafana/grafana-starter-datasource-backend/workflows/CI/badge.svg)](https://github.com/grafana/grafana-datasource-backend/actions?query=workflow%3A%22CI%22)

This is Couchbase datasource plugin for Grafana.
Current version supports querying couchbase clusters using SQL++.

## Installation from sources
Clone this repository and copy `couchbase` directory into your Grafana `plugins` directory.

## SQL++ query requirements
Under the hood plugin wraps user queries into a query with filters on `time` field, which is expected to be provided by the user query:
```sql
SELECT millis_to_str(time) as time, count FROM test
```

