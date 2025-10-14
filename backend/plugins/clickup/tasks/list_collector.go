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
	"fmt"
	"net/http"
	"net/url"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

const RAW_LIST_TABLE = "clickup_api_lists"

var CollectListsMeta = plugin.SubTaskMeta{
	Name:             "collectLists",
	EntryPoint:       CollectLists,
	EnabledByDefault: true,
	Description:      "Collect lists from ClickUp API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectLists(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	// First, collect folderless lists from the space
	taskCtx.GetLogger().Info("Collecting folderless lists from space: %s", data.Options.ScopeId)

	spaceListCollector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_LIST_TABLE,
		},
		ApiClient:   data.ApiClient,
		UrlTemplate: "space/{{ .Params.SpaceId }}/list",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("archived", "false")
			return query, nil
		},
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var result struct {
				Lists []json.RawMessage `json:"lists"`
			}
			err := helper.UnmarshalResponse(res, &result)
			if err != nil {
				return nil, err
			}
			taskCtx.GetLogger().Info("Received %d folderless lists from space", len(result.Lists))
			return result.Lists, nil
		},
	})

	if err != nil {
		return err
	}

	err = spaceListCollector.Execute()
	if err != nil {
		return err
	}

	// Then, collect lists from each folder
	for _, folderId := range data.FolderIds {
		taskCtx.GetLogger().Info("Collecting lists from folder: %s", folderId)

		folderUrlTemplate := fmt.Sprintf("folder/%s/list", folderId)

		folderListCollector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
			RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
				Ctx: taskCtx,
				Params: ClickUpApiParams{
					ConnectionId: data.Options.ConnectionId,
					SpaceId:      data.Options.ScopeId,
				},
				Table: RAW_LIST_TABLE,
			},
			ApiClient:   data.ApiClient,
			UrlTemplate: folderUrlTemplate,
			Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
				query := url.Values{}
				query.Set("archived", "false")
				return query, nil
			},
			ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
				var result struct {
					Lists []json.RawMessage `json:"lists"`
				}
				err := helper.UnmarshalResponse(res, &result)
				if err != nil {
					return nil, err
				}
				taskCtx.GetLogger().Info("Received %d lists from folder", len(result.Lists))
				return result.Lists, nil
			},
		})

		if err != nil {
			return err
		}

		err = folderListCollector.Execute()
		if err != nil {
			return err
		}
	}

	return nil
}
