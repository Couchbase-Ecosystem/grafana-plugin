import { DataSourcePlugin } from '@grafana/data';
import { Couchbase } from './datasource';
import { ConfigEditor } from './ConfigEditor';
import { QueryEditor } from './QueryEditor';
import { CouchbaseQuery, CouchbaseOptions } from './types';

export const plugin = new DataSourcePlugin<Couchbase, CouchbaseQuery, CouchbaseOptions>(Couchbase)
  .setConfigEditor(ConfigEditor)
  .setQueryEditor(QueryEditor);
