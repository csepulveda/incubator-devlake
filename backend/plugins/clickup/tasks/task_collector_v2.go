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

// CollectTasksV2 collects all tasks from a space, including those in folders
// This replaces the separate folder/list collection logic
func CollectTasksV2(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)
	spaceId := data.Options.ScopeId

	taskCtx.GetLogger().Info("Starting task collection for space: %s", spaceId)

	// Step 1: Get all folders in the space
	folderIds, err := getFoldersInSpace(taskCtx, data.ApiClient, spaceId)
	if err != nil {
		return err
	}
	taskCtx.GetLogger().Info("Found %d folders in space", len(folderIds))

	// Step 2: Get lists from each folder + folderless lists
	allListIds := make([]string, 0)

	// Get folderless lists from space
	spaceLists, err := getListsFromSpace(taskCtx, data.ApiClient, spaceId)
	if err != nil {
		return err
	}
	allListIds = append(allListIds, spaceLists...)
	taskCtx.GetLogger().Info("Found %d folderless lists", len(spaceLists))

	// Get lists from each folder
	for _, folderId := range folderIds {
		folderLists, err := getListsFromFolder(taskCtx, data.ApiClient, folderId)
		if err != nil {
			taskCtx.GetLogger().Warn(err, "Failed to get lists from folder %s", folderId)
			continue
		}
		allListIds = append(allListIds, folderLists...)
	}

	taskCtx.GetLogger().Info("Total lists to process: %d", len(allListIds))

	// Step 3: Collect tasks from each list using an iterator
	// Create a list of list objects for the iterator
	type ListInput struct {
		ListId string
	}

	iterator := helper.NewQueueIterator()
	for _, listId := range allListIds {
		iterator.Push(&ListInput{ListId: listId})
	}

	collector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: ClickUpApiParams{
			ConnectionId: data.Options.ConnectionId,
			SpaceId:      spaceId,
		},
		Table: RAW_TASK_TABLE,
	})

	if err != nil {
		return err
	}

	err = collector.InitCollector(helper.ApiCollectorArgs{
		ApiClient:   data.ApiClient,
		PageSize:    100,
		Concurrency: 1, // Use sequential collection to avoid rate limiting (ClickUp: 100 req/min)
		Input:       iterator,
		UrlTemplate: "list/{{ .Input.ListId }}/task",
		Query: func(reqData *helper.RequestData) (url.Values, errors.Error) {
			query := url.Values{}
			query.Set("page", fmt.Sprintf("%d", reqData.Pager.Page))
			query.Set("include_closed", "true")
			query.Set("subtasks", "true")

			// TimeAfter is automatically provided by the stateful collector from blueprint
			// Access it via GetSince() or check reqData.CustomData if needed
			if collector.GetSince() != nil {
				query.Set("date_updated_gt", fmt.Sprintf("%d", collector.GetSince().UnixMilli()))
				taskCtx.GetLogger().Info("Using incremental collection from: %s", collector.GetSince().Format("2006-01-02"))
			} else {
				taskCtx.GetLogger().Info("Performing full collection (no TimeAfter)")
			}

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
			taskCtx.GetLogger().Info("Received %d tasks", len(result.Tasks))
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

	taskCtx.GetLogger().Info("Task collection completed")
	return nil
}

func getFoldersInSpace(taskCtx plugin.SubTaskContext, apiClient *helper.ApiAsyncClient, spaceId string) ([]string, errors.Error) {
	url := fmt.Sprintf("space/%s/folder", spaceId)
	res, err := apiClient.Get(url, nil, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Folders []struct {
			Id       string `json:"id"`
			Archived bool   `json:"archived"`
		} `json:"folders"`
	}

	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}

	folderIds := make([]string, 0)
	for _, folder := range result.Folders {
		if !folder.Archived {
			folderIds = append(folderIds, folder.Id)
		}
	}

	return folderIds, nil
}

func getListsFromSpace(taskCtx plugin.SubTaskContext, apiClient *helper.ApiAsyncClient, spaceId string) ([]string, errors.Error) {
	url := fmt.Sprintf("space/%s/list", spaceId)
	res, err := apiClient.Get(url, nil, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Lists []struct {
			Id       string `json:"id"`
			Archived bool   `json:"archived"`
		} `json:"lists"`
	}

	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}

	listIds := make([]string, 0)
	for _, list := range result.Lists {
		if !list.Archived {
			listIds = append(listIds, list.Id)
		}
	}

	return listIds, nil
}

func getListsFromFolder(taskCtx plugin.SubTaskContext, apiClient *helper.ApiAsyncClient, folderId string) ([]string, errors.Error) {
	url := fmt.Sprintf("folder/%s/list", folderId)
	res, err := apiClient.Get(url, nil, nil)
	if err != nil {
		return nil, err
	}

	var result struct {
		Lists []struct {
			Id       string `json:"id"`
			Archived bool   `json:"archived"`
		} `json:"lists"`
	}

	err = helper.UnmarshalResponse(res, &result)
	if err != nil {
		return nil, err
	}

	listIds := make([]string, 0)
	for _, list := range result.Lists {
		if !list.Archived {
			listIds = append(listIds, list.Id)
		}
	}

	return listIds, nil
}
