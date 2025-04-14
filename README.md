# Couchbase datasource plugin for Grafana

!["Couchbase-powered dashboard"](res/dashboards.png "Example dashboard")

This is a simple community-supported Grafana datasource plugin that allows querying time series and log data from Couchbase Clusters, inclding Capella clusters.

## Installation from sources
This plugin uses [Standard Grafana DataSource Backend Plugin build process](https://grafana.com/developers/plugin-tools/tutorials/build-a-data-source-backend-plugin). Also, the `./run.sh` script in this repository is used to launch Grafana with the plugin in a Docker container and usually contains the latest build process commands.

* Clone this repository 
* In `couchbase-datasource` directory, run `yarn install && yarn build && mage -v`
* Copy `couchbase-datasource` directory into your Grafana `plugins` directory.
* Set the following environment variables (or edit grafana configuration file according to documentation):
    - "GF_PLUGINS_ALLOW_LOADING_UNSIGNED_PLUGINS=couchbase-datasource"
    - "GF_PLUGIN_APP_TLS_SKIP_VERIFY_INSECURE=true"
* Restart grafana
* Add a new datasource in configuration and configure it with your cluster information.

## Usage
The datasource supports only `SELECT` statements.
The datasource plugin provides two additional sql++ `WHERE` clause functions that inject into all submitted queries time range filtering clauses according to 
selected in Grafana UI report time range:
- `str_time_range(<fieldname>)` for filtering on RFC3339 dates
- `time_range(<fieldname>)` for filtering on millisecond timestamps

Both functions take the name of the field to be used for filtering. 
One and only one of these functions *must* be included in every query submitted through the plugin:
Examples:

```sql
select count, time_string from test where str_time_range(time_string)
select count, timestamp from test where time_range(timestamp)
```

These are pseudo-functions, references to them are replaced with a set of `WHERE` filters on provided field by the plugin before the query is sent to the cluster.


## Development instructions 
Add `datasources/couchbase.yaml` with your datasource configuration:
```yaml
apiVersion: 1
datasources:
- name: Couchbase
  type: couchbase-datasource
  access: proxy
  jsonData:
    host: <...>
    username: <...>
  secureJsonData:
    password: <...>
```

Use `./run.sh` to start grafana in docker container with following mounts:
```yaml
    volumes:
    - ./couchbase:/var/lib/grafana/plugins/couchbase
    - ./datasources:/etc/grafana/provisioning/datasources
```

Open grafana at http://localhost:3000, use `admin` as both login and password. 
You don't need to setup a new password after you login despite grafana asking you to do that -- just reload the page.

---

## 📢 Support Policy

We truly appreciate your interest in this project!  
This project is **community-maintained**, which means it's **not officially supported** by our support team.

If you need help, have found a bug, or want to contribute improvements, the best place to do that is right here — by [opening a GitHub issue](https://github.com/Couchbase-Ecosystem/grafana-plugin/issues).  
Our support portal is unable to assist with requests related to this project, so we kindly ask that all inquiries stay within GitHub.

Your collaboration helps us all move forward together — thank you!
