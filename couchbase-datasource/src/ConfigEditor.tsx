import React, { ChangeEvent, PureComponent } from 'react';
import { LegacyForms } from '@grafana/ui';
import { DataSourcePluginOptionsEditorProps } from '@grafana/data';
import { CouchbaseOptions, CouchbaseSecureData } from './types';

const { SecretFormField, FormField } = LegacyForms;

interface Props extends DataSourcePluginOptionsEditorProps<CouchbaseOptions> {}

interface State {}

export class ConfigEditor extends PureComponent<Props, State> {
  onHostChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        host: event.target.value,
      },
    });
  };

  onUserChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      jsonData: {
        ...options.jsonData,
        username: event.target.value,
      },
    });
  };

  // Secure field (only sent to the backend)
  onPasswordChange = (event: ChangeEvent<HTMLInputElement>) => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        ...options.secureJsonData,
        password: event.target.value,
      },
    });
  };

  onPasswordReset = () => {
    const { onOptionsChange, options } = this.props;
    onOptionsChange({
      ...options,
      secureJsonData: {
        password: "",
      },
    });
  };

  render() {
    const { options } = this.props;
    const { jsonData, secureJsonFields } = options;
    const secureJsonData = (options.secureJsonData || {}) as CouchbaseSecureData;

    return (
      <div className="gf-form-group">
        <div className="gf-form">
          <FormField
            label="Cluster"
            labelWidth={10}
            inputWidth={20}
            onChange={this.onHostChange}
            value={jsonData.host || ''}
            placeholder="Couchbase cluster address"
          />
        </div>
        <div className="gf-form">
          <FormField
            label="Username"
            labelWidth={10}
            inputWidth={20}
            onChange={this.onUserChange}
            value={jsonData.username || ''}
            placeholder="Couchbase cluster username"
          />
        </div>
        <div className="gf-form">
          <SecretFormField
            isConfigured={(secureJsonFields && secureJsonFields.password) as boolean}
            value={secureJsonData.password || ''}
            label="Password"
            placeholder="Couchbase cluster password"
            labelWidth={10}
            inputWidth={20}
            onReset={this.onPasswordReset}
            onChange={this.onPasswordChange}
          />
        </div>
      </div>
    );
  }
}
