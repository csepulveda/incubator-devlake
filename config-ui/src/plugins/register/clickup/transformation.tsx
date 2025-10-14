/*
 * Licensed to the Apache Software Foundation (ASF) under one or more
 * contributor license agreements.  See the NOTICE file distributed with
 * this work for additional information regarding copyright ownership.
 * The ASF licenses this file to You under the Apache License, Version 2.0
 * (the "License"); you may not use this file except in compliance with
 * the License.  You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

import { useState, useEffect } from 'react';
import { Form, Select, Alert } from 'antd';

import { ExternalLink } from '@/components';
import { DOC_URL } from '@/release';

interface Props {
  connectionId: ID;
  scopeId?: ID;
  transformation: any;
  setTransformation: React.Dispatch<React.SetStateAction<any>>;
}

export const ClickUpTransformation = ({ connectionId, scopeId, transformation, setTransformation }: Props) => {
  const [folders, setFolders] = useState<Array<{ id: string; name: string }>>([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState<string | null>(null);

  useEffect(() => {
    if (!scopeId) {
      setError('No space selected. Please select a Data Scope first.');
      return;
    }

    const fetchFolders = async () => {
      setLoading(true);
      setError(null);
      try {
        // Fetch folders from the ClickUp API through DevLake proxy
        const response = await fetch(`/api/plugins/clickup/connections/${connectionId}/proxy/space/${scopeId}/folder`);

        if (!response.ok) {
          throw new Error(`API returned ${response.status}`);
        }

        const data = await response.json();

        if (data.folders) {
          const activeFolders = data.folders
            .filter((f: any) => !f.archived)
            .map((f: any) => ({
              id: f.id,
              name: f.name,
            }));

          setFolders(activeFolders);

          if (activeFolders.length === 0) {
            setError('No active folders found in this space.');
          }
        } else {
          setFolders([]);
        }
      } catch (err) {
        console.error('Failed to fetch folders:', err);
        setError('Failed to load folders. Please check your connection.');
        setFolders([]);
      } finally {
        setLoading(false);
      }
    };

    fetchFolders();
  }, [connectionId, scopeId]);

  const handleChangeFolders = (value: string[]) => {
    setTransformation({
      ...transformation,
      folderIds: value,
    });
  };

  return (
    <>
      <Alert
        style={{ marginBottom: 16 }}
        message="By default, ClickUp plugin collects tasks from all folders in the selected space. You can optionally select specific folders to collect tasks from."
        type="info"
      />

      {!scopeId ? (
        <Alert
          style={{ marginBottom: 16 }}
          message="This scope config can be used with any ClickUp space. Folder selection will be available when you associate this config with a specific Data Scope. Leave this config as-is to collect from all folders."
          type="info"
          showIcon
        />
      ) : error ? (
        <Alert
          style={{ marginBottom: 16 }}
          message={error}
          type="warning"
          showIcon
        />
      ) : null}

      {scopeId && !error && (
        <Form.Item label="Folders" tooltip="Leave empty to collect from all folders in the space">
          <Select
            mode="multiple"
            placeholder="Select specific folders (optional)"
            loading={loading}
            value={transformation.folderIds || []}
            onChange={handleChangeFolders}
            options={folders.map((folder) => ({
              label: folder.name,
              value: folder.id,
            }))}
            allowClear
            notFoundContent={loading ? 'Loading...' : 'No folders found'}
          />
        </Form.Item>
      )}

      <Form.Item>
        <p style={{ color: '#666', fontSize: '14px' }}>
          The ClickUp plugin automatically links tasks to GitHub PRs by:
        </p>
        <ul style={{ color: '#666', fontSize: '14px', paddingLeft: '20px' }}>
          <li>Matching task IDs in PR titles or branch names</li>
          <li>Finding PR URLs in task comments</li>
        </ul>
        <ExternalLink link={DOC_URL.PLUGIN.CLICKUP?.BASIS || 'https://devlake.apache.org'}>
          Learn more about ClickUp configuration
        </ExternalLink>
      </Form.Item>
    </>
  );
};
