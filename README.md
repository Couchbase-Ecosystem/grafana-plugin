# grafana-plugin
Grafana Couchbase datasource plugin

UNDER DEVELOPMENT

## Installation from sources
Clone this repository and copy `couchbase` directory into your Grafana `plugins` directory.

## SQL++ query requirements
Under the hood plugin wraps user queries into a query with filters on `time` field, which is expected to be provided by the user query:
```sql
SELECT millis_to_str(time) as time, count FROM test
```

## Development instructions 
Add `datasources/couchbase.yaml` with your datasource configuration:
```yaml
apiVersion: 1
datasources:
- name: Couchbase
  type: couchbase
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
