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

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

var ExtractListsMeta = plugin.SubTaskMeta{
	Name:             "extractLists",
	EntryPoint:       ExtractLists,
	EnabledByDefault: true,
	Description:      "Extract lists from ClickUp API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type ClickUpList struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	SpaceId  string `json:"space_id"`
	Archived bool   `json:"archived"`
}

func ExtractLists(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_LIST_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var list ClickUpList
			err := json.Unmarshal(row.Data, &list)
			if err != nil {
				return nil, errors.Default.Wrap(err, "error unmarshaling list")
			}

			taskCtx.GetLogger().Info("Extracted list: %s (ID: %s)", list.Name, list.Id)

			// Store list IDs in memory for task collection
			if data.ListIds == nil {
				data.ListIds = make([]string, 0)
			}
			data.ListIds = append(data.ListIds, list.Id)

			taskCtx.GetLogger().Info("Total lists collected so far: %d", len(data.ListIds))

			// We don't need to store lists in the database, just collect the IDs
			return nil, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}
