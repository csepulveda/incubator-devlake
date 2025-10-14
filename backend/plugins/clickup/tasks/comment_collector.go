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
	"reflect"
	"regexp"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

const RAW_COMMENT_TABLE = "clickup_api_comments"

var CollectCommentsMeta = plugin.SubTaskMeta{
	Name:             "collectComments",
	EntryPoint:       CollectComments,
	EnabledByDefault: true,
	Description:      "Collect comments from ClickUp tasks to extract PR links",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func CollectComments(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)
	db := taskCtx.GetDal()

	// Get task IDs that need comment collection
	// Strategy: Only collect comments for tasks that:
	// 1. Don't have a PR associated yet (git_hub_pr_number = 0 or NULL), OR
	// 2. Were updated recently (to catch new comments on existing PR-linked tasks)
	// This avoids re-collecting all comments on every run

	// Get the "since" time from the collector (incremental sync)
	collector, err := helper.NewStatefulApiCollector(helper.RawDataSubTaskArgs{
		Ctx: taskCtx,
		Params: ClickUpApiParams{
			ConnectionId: data.Options.ConnectionId,
			SpaceId:      data.Options.ScopeId,
		},
		Table: RAW_COMMENT_TABLE,
	})
	if err != nil {
		return err
	}

	var clauses []dal.Clause
	if collector.GetSince() != nil {
		// Incremental: only tasks updated since last sync
		taskCtx.GetLogger().Info("Incremental comment collection: tasks updated since %s", collector.GetSince().Format("2006-01-02"))
		clauses = []dal.Clause{
			dal.From(&models.ClickUpTask{}),
			dal.Select("id"),
			dal.Where("connection_id = ? AND date_updated >= ?", data.Options.ConnectionId, collector.GetSince()),
		}
	} else {
		// Full sync: only tasks without PR links (optimization for first run)
		taskCtx.GetLogger().Info("Full comment collection: all tasks without PR links")
		clauses = []dal.Clause{
			dal.From(&models.ClickUpTask{}),
			dal.Select("id"),
			dal.Where("connection_id = ? AND (git_hub_pr_number IS NULL OR git_hub_pr_number = 0)", data.Options.ConnectionId),
		}
	}

	cursor, err := db.Cursor(clauses...)
	if err != nil {
		return err
	}
	defer cursor.Close()

	// Collect comments for each task
	iterator, err := helper.NewDalCursorIterator(db, cursor, reflect.TypeOf(models.ClickUpTask{}))
	if err != nil {
		return err
	}

	apiCollector, err := helper.NewApiCollector(helper.ApiCollectorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_COMMENT_TABLE,
		},
		ApiClient: data.ApiClient,
		Input:     iterator,
		UrlTemplate: "task/{{ .Input.Id }}/comment",
		ResponseParser: func(res *http.Response) ([]json.RawMessage, errors.Error) {
			var result struct {
				Comments []json.RawMessage `json:"comments"`
			}
			err := helper.UnmarshalResponse(res, &result)
			if err != nil {
				return nil, err
			}

			// Filter comments: only keep those with GitHub or GitLab PR URLs
			// This saves storage space and processing time
			// Match with or without https:// prefix
			prRegex := regexp.MustCompile(`(https?://)?(?:www\.)?(github|gitlab)\.com/[^/\s]+/[^/\s]+/(pull|merge_requests)/\d+`)

			filteredComments := make([]json.RawMessage, 0)
			totalComments := len(result.Comments)

			for _, comment := range result.Comments {
				// Parse comment to check if it contains a PR URL
				var commentData struct {
					CommentText string `json:"comment_text"`
				}
				if err := json.Unmarshal(comment, &commentData); err == nil {
					if prRegex.MatchString(commentData.CommentText) {
						filteredComments = append(filteredComments, comment)
					}
				}
			}

			// Log with INFO level so it shows in pipeline logs
			if totalComments > 0 {
				taskCtx.GetLogger().Info("Task has %d comments, %d with PR URLs", totalComments, len(filteredComments))
			}

			return filteredComments, nil
		},
	})

	if err != nil {
		return err
	}

	return apiCollector.Execute()
}
