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
	"regexp"
	"strconv"
	"strings"

	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

var ExtractCommentsMeta = plugin.SubTaskMeta{
	Name:             "extractComments",
	EntryPoint:       ExtractComments,
	EnabledByDefault: true,
	Description:      "Extract GitHub/GitLab PR URLs from ClickUp task comments",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type ClickUpApiComment struct {
	Id          string `json:"id"`
	CommentText string `json:"comment_text"`
	Date        string `json:"date"`
}

func ExtractComments(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)
	db := taskCtx.GetDal()

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_COMMENT_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiComment ClickUpApiComment
			err := errors.Convert(json.Unmarshal(row.Data, &apiComment))
			if err != nil {
				return nil, err
			}

			// Extract task ID from raw data input
			var input struct {
				Id string `json:"Id"`
			}
			err = errors.Convert(json.Unmarshal(row.Input, &input))
			if err != nil {
				return nil, err
			}

			// Look for GitHub or GitLab PR/MR URLs in comment text
			// Matches with or without https:// prefix:
			//   - GitHub: github.com/org/repo/pull/123 or https://github.com/org/repo/pull/123
			//   - GitLab: gitlab.com/org/repo/merge_requests/456 or https://gitlab.com/org/repo/merge_requests/456
			githubRegex := regexp.MustCompile(`(?:https?://)?(?:www\.)?github\.com/([^/\s]+)/([^/\s]+)/pull/(\d+)`)
			gitlabRegex := regexp.MustCompile(`(?:https?://)?(?:www\.)?gitlab\.com/([^/\s]+)/([^/\s]+)/merge_requests/(\d+)`)

			var prUrl string
			var prNumber int

			// Try GitHub first
			if matches := githubRegex.FindStringSubmatch(apiComment.CommentText); len(matches) > 3 {
				prNumber, _ = strconv.Atoi(matches[3])
				prUrl = matches[0]
				// Remove "/files" suffix if present
				if len(prUrl) > 6 && prUrl[len(prUrl)-6:] == "/files" {
					prUrl = prUrl[:len(prUrl)-6]
				}
				// Normalize URL: add https:// if missing
				if !strings.HasPrefix(prUrl, "http://") && !strings.HasPrefix(prUrl, "https://") {
					prUrl = "https://" + prUrl
				}
			} else if matches := gitlabRegex.FindStringSubmatch(apiComment.CommentText); len(matches) > 3 {
				// Try GitLab
				prNumber, _ = strconv.Atoi(matches[3])
				prUrl = matches[0]
				// Normalize URL: add https:// if missing
				if !strings.HasPrefix(prUrl, "http://") && !strings.HasPrefix(prUrl, "https://") {
					prUrl = "https://" + prUrl
				}
			}

			if prNumber > 0 {
				// Found a PR/MR URL in the comment
				// Update the task with PR information
				task := &models.ClickUpTask{}
				err := db.First(
					task,
					dal.From(&models.ClickUpTask{}),
					dal.Where("connection_id = ? AND id = ?", data.Options.ConnectionId, input.Id),
				)

				if err == nil && task.GitHubPRNumber == 0 {
					// Only update if not already set
					task.GitHubPRUrl = prUrl
					task.GitHubPRNumber = prNumber

					err = db.Update(task)
					if err != nil {
						taskCtx.GetLogger().Warn(err, "failed to update task with PR from comment")
					}
				}
			}

			return []interface{}{}, nil // We update tasks directly, no need to return data
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}
