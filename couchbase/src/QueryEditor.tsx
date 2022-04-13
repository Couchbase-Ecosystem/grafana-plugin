import React, { PureComponent } from 'react';
import { QueryField } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MyDataSourceOptions, MyQuery } from './types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  onQueryChange = (value: string) => {
    this.props.onChange({
      ...this.props.query,
      query: value,
    });
  };

  render() {
    return <QueryField portalOrigin="couchbase" onChange={this.onQueryChange} />;
  }
}
