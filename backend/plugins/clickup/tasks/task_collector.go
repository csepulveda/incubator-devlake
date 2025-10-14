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

const RAW_TASK_TABLE = "clickup_api_tasks"

var CollectTasksMeta = plugin.SubTaskMeta{
	Name:             "collectTasks",
	EntryPoint:       CollectTasks,
	EnabledByDefault: true,
	Description:      "Collect tasks from ClickUp API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectTasks(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	// Check if we have list IDs from the list collector
	if len(data.ListIds) == 0 {
		taskCtx.GetLogger().Info("No lists found in space %s, skipping task collection", data.Options.ScopeId)
		return nil
	}

	// Collect tasks from each list
	for _, listId := range data.ListIds {
		taskCtx.GetLogger().Info("Collecting tasks from list: %s", listId)

		// Create URL with the list ID directly
		urlTemplate := fmt.Sprintf("list/%s/task", listId)

		collector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
			RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
				Ctx: taskCtx,
				Params: ClickUpApiParams{
					ConnectionId: data.Options.ConnectionId,
					SpaceId:      data.Options.ScopeId,
				},
				Table: RAW_TASK_TABLE,
			},
			ApiClient:   data.ApiClient,
			PageSize:    100,
			UrlTemplate: urlTemplate,
			Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
				query := url.Values{}
				query.Set("page", fmt.Sprintf("%d", reqData.Pager.Page))
				query.Set("include_closed", "true")
				query.Set("subtasks", "true")

				// Note: This old collector doesn't support TimeAfter filtering
				// Use CollectTasksV2 instead for incremental collection

				return query, nil
			},
			ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
				var result struct {
					Tasks []json.RawMessage `json:"tasks"`
				}
				err := helper.UnmarshalResponse(res, &result)
				if err != nil {
					return nil, err
				}
				taskCtx.GetLogger().Info("Received %d tasks from list %s", len(result.Tasks), listId)
				if len(result.Tasks) > 0 {
					taskCtx.GetLogger().Info("First task sample: %s", string(result.Tasks[0][:100]))
				}
				return result.Tasks, nil
			},
		})

		if err != nil {
			return err
		}

		err = collector.Execute()
		if err != nil {
			return err
		}
	}

	return nil
}

type ClickUpApiParams struct {
	ConnectionId uint64 `json:"ConnectionId"`
	SpaceId      string `json:"SpaceId"`
}
