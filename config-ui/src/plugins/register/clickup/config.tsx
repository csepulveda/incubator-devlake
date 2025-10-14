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

import { IPluginConfig } from '@/types';

import Icon from './assets/icon.svg?react';

export const ClickUpConfig: IPluginConfig = {
  plugin: 'clickup',
  name: 'ClickUp',
  icon: ({ color }) => <Icon fill={color} />,
  sort: 3,
  connection: {
    docLink: 'https://docs.clickup.com/en/articles/1367130-getting-started-with-the-clickup-api',
    fields: [
      'name',
      {
        key: 'endpoint',
        subLabel: 'Provide the ClickUp API endpoint.',
        placeholder: 'https://api.clickup.com/api/v2',
        defaultValue: 'https://api.clickup.com/api/v2',
      },
      {
        key: 'token',
        label: 'Personal API Token',
        subLabel: 'Your ClickUp Personal API Token (starts with pk_)',
        placeholder: 'pk_...',
        type: 'password',
      },
      'proxy',
      {
        key: 'rateLimitPerHour',
        subLabel:
          'By default, DevLake uses 100 requests/hour for data collection for ClickUp. You can adjust the collection speed by setting up your desirable rate limit.',
        defaultValue: 100,
      },
    ],
  },
  dataScope: {
    title: 'Tasks',
  },
  scopeConfig: {
    entities: ['TICKET'],
    transformation: {},
  },
};
