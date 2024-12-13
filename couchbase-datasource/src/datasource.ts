import { DataSourceInstanceSettings, ScopedVars } from '@grafana/data';
import { DataSourceWithBackend, getTemplateSrv } from '@grafana/runtime';
import { CouchbaseOptions, CouchbaseQuery } from './types';

export class Couchbase extends DataSourceWithBackend<CouchbaseQuery, CouchbaseOptions> {
  annotations = {};
  constructor(instanceSettings: DataSourceInstanceSettings<CouchbaseOptions>) {
    super(instanceSettings);
  }

  // Support template variables
  applyTemplateVariables(query: CouchbaseQuery, scopedVars: ScopedVars) {
    return {
      ...query,
      query: getTemplateSrv().replace(query.query, scopedVars)
    };
  }
}
