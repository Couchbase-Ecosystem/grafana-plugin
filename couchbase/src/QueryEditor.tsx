import React, { PureComponent } from 'react';
import { FieldSet, Switch, InlineField, QueryField, InlineFieldRow } from '@grafana/ui';
import { QueryEditorProps } from '@grafana/data';
import { DataSource } from './datasource';
import { MyDataSourceOptions, MyQuery } from './types';

type Props = QueryEditorProps<DataSource, MyQuery, MyDataSourceOptions>;

export class QueryEditor extends PureComponent<Props> {
  onFtsChange = () => {
    this.props.onChange({
      ...this.props.query,
      fts: !this.props.query.fts,
    });
  };

  onQueryChange = (value: string) => {
    this.props.onChange({
      ...this.props.query,
      query: value,
    });
  };

  render() {
    const style = {
      alignItems: 'center',
    } as React.CSSProperties;

    return (
      <FieldSet>
        <InlineFieldRow>
          <InlineField label="FTS" labelWidth={10} style={style}>
            <Switch checked={this.props.query.fts || false} label="FTS Query" onChange={this.onFtsChange} />
          </InlineField>
          <InlineField grow>
            <QueryField portalOrigin="couchbase" onChange={this.onQueryChange} />
          </InlineField>
        </InlineFieldRow>
      </FieldSet>
    );
  }
}
