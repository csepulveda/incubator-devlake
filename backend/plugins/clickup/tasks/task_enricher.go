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
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

var EnrichTasksWithGitHubMeta = plugin.SubTaskMeta{
	Name:             "enrichTasksWithGitHub",
	EntryPoint:       EnrichTasksWithGitHub,
	EnabledByDefault: true,
	Description:      "Enrich ClickUp tasks with GitHub PR and branch information",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

func EnrichTasksWithGitHub(taskCtx plugin.SubTaskContext) errors.Error {
	db := taskCtx.GetDal()
	data := taskCtx.GetData().(*ClickUpTaskData)

	// Get all tasks with GitHub PR URLs (note: field name has underscores)
	cursor, err := db.Cursor(
		dal.From(&models.ClickUpTask{}),
		dal.Where("connection_id = ? AND git_hub_pr_url != ''", data.Options.ConnectionId),
	)
	if err != nil {
		return err
	}
	defer cursor.Close()

	enrichedCount := 0

	// For each task with a PR URL, enrich with branch info from pull_requests table
	for cursor.Next() {
		var task models.ClickUpTask
		err = db.Fetch(cursor, &task)
		if err != nil {
			return err
		}

		// Query pull_requests table to get branch name
		// The PR URL format is: https://github.com/org/repo/pull/123
		// We need to match by pull_request_key (not number)
		if task.GitHubPRNumber > 0 {
			var prBranch struct {
				HeadRef string
			}

			err = db.First(
				&prBranch,
				dal.From("pull_requests"),
				dal.Select("head_ref"),
				dal.Where("pull_request_key = ?", task.GitHubPRNumber),
			)

			if err == nil && prBranch.HeadRef != "" {
				task.GitHubBranch = prBranch.HeadRef

				// Update the task with branch info
				err = db.Update(&task)
				if err != nil {
					taskCtx.GetLogger().Warn(err, "failed to update task with branch info")
				} else {
					enrichedCount++
					taskCtx.GetLogger().Info("Enriched task %s with branch: %s", task.Id, prBranch.HeadRef)
				}
			} else {
				taskCtx.GetLogger().Debug("No PR found for task %s (PR #%d)", task.Id, task.GitHubPRNumber)
			}
		}
	}

	taskCtx.GetLogger().Info("Enriched %d tasks with GitHub branch info", enrichedCount)

	return nil
}
