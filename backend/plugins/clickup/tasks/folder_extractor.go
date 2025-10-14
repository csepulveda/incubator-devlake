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

var ExtractFoldersMeta = plugin.SubTaskMeta{
	Name:             "extractFolders",
	EntryPoint:       ExtractFolders,
	EnabledByDefault: true,
	Description:      "Extract folders from ClickUp API",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type ClickUpFolder struct {
	Id       string `json:"id"`
	Name     string `json:"name"`
	Hidden   bool   `json:"hidden"`
	SpaceId  string `json:"space_id"`
	Archived bool   `json:"archived"`
}

func ExtractFolders(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_FOLDER_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var folder ClickUpFolder
			err := json.Unmarshal(row.Data, &folder)
			if err != nil {
				return nil, errors.Default.Wrap(err, "error unmarshaling folder")
			}

			taskCtx.GetLogger().Info("Extracted folder: %s (ID: %s)", folder.Name, folder.Id)

			// Store folder IDs in memory for list collection
			if data.FolderIds == nil {
				data.FolderIds = make([]string, 0)
			}
			data.FolderIds = append(data.FolderIds, folder.Id)

			taskCtx.GetLogger().Info("Total folders collected so far: %d", len(data.FolderIds))

			// We don't need to store folders in the database, just collect the IDs
			return nil, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}
