import React, { PureComponent, FormEvent } from 'react';
import { FieldSet, InlineFieldRow, QueryField, InlineSwitch } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { Couchbase } from './datasource';
import { CouchbaseOptions, CouchbaseQuery, defaultQuery } from './types';
import { defaults } from 'lodash';

type Props = QueryEditorProps<Couchbase, CouchbaseQuery, CouchbaseOptions>;

export class QueryEditor extends PureComponent<Props> {
  onQueryChange = (value: string) => {
    this.props.onChange({
      ...this.props.query,
      query: value,
    });
  };

  onAnalyticsChange = (e: FormEvent<HTMLInputElement>) => {
    this.props.onChange({
      ...this.props.query,
      analytics: e.currentTarget.checked,
    });
  };

  render() {
    const query = defaults(this.props.query, defaultQuery);

    return (
      <FieldSet>
        <InlineFieldRow>
          <QueryField query={query.query} portalOrigin="couchbase" onChange={this.onQueryChange} />
        </InlineFieldRow>
        <div style={{ display: 'flex', justifyContent: 'flex-end' }}>
          <InlineSwitch
            label="Run using analytics service"
            showLabel={true}
            defaultChecked={query.analytics}
            value={query.analytics}
            onChange={this.onAnalyticsChange}
          />
        </div>
      </FieldSet>
    );
  }
}
