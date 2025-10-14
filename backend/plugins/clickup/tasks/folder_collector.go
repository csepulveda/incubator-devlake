/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package tasks

import (
	"encoding/json"
	"net/http"
	"net/url"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const RAW_FOLDER_TABLE = "clickup_api_folders"

var CollectFoldersMeta = plugin.SubTaskMeta{
	Name:             "collectFolders",
	EntryPoint:       CollectFolders,
	EnabledByDefault: true,
	Description:      "Collect folders from ClickUp API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectFolders(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	taskCtx.GetLogger().Info("Collecting folders from space: %s", data.Options.ScopeId)

	collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_FOLDER_TABLE,
		},
		ApiClient:   data.ApiClient,
		UrlTemplate: "space/{{ .Params.SpaceId }}/folder",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("archived", "false")
			return query, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var result struct {
				Folders []json.RawMessage `json:"folders"`
			}
			err := helper.UnmarshalResponse(res, &result)
			if err != nil {
				return nil, err
			}
			taskCtx.GetLogger().Info("Received %d folders from API", len(result.Folders))
			return result.Folders, nil
		},
	})

	if err != nil {
		return err
	}

	return collector.Execute()
}
