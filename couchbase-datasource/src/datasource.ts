import { DataSourceInstanceSettings } from '@grafana/data';
import { DataSourceWithBackend } from '@grafana/runtime';
import { CouchbaseOptions, CouchbaseQuery } from './types';

export class Couchbase extends DataSourceWithBackend<CouchbaseQuery, CouchbaseOptions> {
  annotations = {};
  constructor(instanceSettings: DataSourceInstanceSettings<CouchbaseOptions>) {
    super(instanceSettings);
  }
}
