import { DataQuery, DataSourceJsonData } from '@grafana/data';

export interface MyQuery extends DataQuery {
  query: string;
  analytics: boolean;
}

export const defaultQuery: Partial<MyQuery> = {
  query: '',
  analytics: false,
};

export interface SelectField {
  label?: string;
  expression: SqlExpression;
}

export interface SqlExpression {}

/**
 * These are options configured for each DataSource instance.
 */
export interface MyDataSourceOptions extends DataSourceJsonData {
  host?: string;
  username?: string;
  bucket?: string;
}

/**
 * Value that is used in the backend, but never sent over HTTP to the frontend
 */
export interface MySecureJsonData {
  password?: string;
}
