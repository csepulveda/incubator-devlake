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
	"strconv"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

var ExtractTasksMeta = plugin.SubTaskMeta{
	Name:             "extractTasks",
	EntryPoint:       ExtractTasks,
	EnabledByDefault: true,
	Description:      "Extract ClickUp tasks from raw data",
	DomainTypes:      []string{plugin.DOMAIN_TYPE_TICKET},
}

type ClickUpApiTask struct {
	Id          string `json:"id"`
	Name        string `json:"name"`
	// Description field intentionally omitted - not stored in DB, only comments are used
	Status      struct {
		Status string `json:"status"`
	} `json:"status"`
	Priority *struct {
		Priority string `json:"priority"`
	} `json:"priority"`
	List struct {
		Id string `json:"id"`
	} `json:"list"`
	Folder struct {
		Id string `json:"id"`
	} `json:"folder"`
	Space struct {
		Id string `json:"id"`
	} `json:"space"`
	DateCreated string  `json:"date_created"`
	DateUpdated string  `json:"date_updated"`
	DateDone    *string `json:"date_done"`
	DateClosed  *string `json:"date_closed"`
	StartDate   *string `json:"start_date"`
	DueDate     *string `json:"due_date"`
	Creator     struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
	} `json:"creator"`
	Assignees []struct {
		Id       int    `json:"id"`
		Username string `json:"username"`
	} `json:"assignees"`
	Url string `json:"url"`
}

func ExtractTasks(taskCtx plugin.SubTaskContext) errors.Error {
	data := taskCtx.GetData().(*ClickUpTaskData)

	extractor, err := helper.NewApiExtractor(helper.ApiExtractorArgs{
		RawDataSubTaskArgs: helper.RawDataSubTaskArgs{
			Ctx: taskCtx,
			Params: ClickUpApiParams{
				ConnectionId: data.Options.ConnectionId,
				SpaceId:      data.Options.ScopeId,
			},
			Table: RAW_TASK_TABLE,
		},
		Extract: func(row *helper.RawData) ([]interface{}, errors.Error) {
			var apiTask ClickUpApiTask
			err := errors.Convert(json.Unmarshal(row.Data, &apiTask))
			if err != nil {
				return nil, err
			}

			// Truncate name to 255 chars to fit database field
			taskName := apiTask.Name
			if len(taskName) > 255 {
				taskName = taskName[:255]
			}

			task := &models.ClickUpTask{
				ConnectionId: data.Options.ConnectionId,
				Id:           apiTask.Id,
				Name:         taskName,
				// Description intentionally omitted - we only need comments for PR linking
				Status:       apiTask.Status.Status,
				ListId:       apiTask.List.Id,
				FolderId:     apiTask.Folder.Id,
				SpaceId:      apiTask.Space.Id,
				Creator:      apiTask.Creator.Username,
				Url:          apiTask.Url,
			}

			// Parse priority
			if apiTask.Priority != nil {
				task.Priority = apiTask.Priority.Priority
			}

			// Parse dates
			if createdAt, err := parseClickUpTime(apiTask.DateCreated); err == nil {
				task.DateCreated = createdAt
			}
			if updatedAt, err := parseClickUpTime(apiTask.DateUpdated); err == nil {
				task.DateUpdated = updatedAt
			}
			if apiTask.DateDone != nil {
				if dateDone, err := parseClickUpTime(*apiTask.DateDone); err == nil {
					task.DateDone = &dateDone
				}
			}
			if apiTask.DateClosed != nil {
				if dateClosed, err := parseClickUpTime(*apiTask.DateClosed); err == nil {
					task.DateClosed = &dateClosed
				}
			}
			if apiTask.StartDate != nil {
				if startDate, err := parseClickUpTime(*apiTask.StartDate); err == nil {
					task.StartDate = &startDate
				}
			}
			if apiTask.DueDate != nil {
				if dueDate, err := parseClickUpTime(*apiTask.DueDate); err == nil {
					task.DueDate = &dueDate
				}
			}

			// Parse assignees
			assigneeIds := make([]string, 0, len(apiTask.Assignees))
			for _, assignee := range apiTask.Assignees {
				assigneeIds = append(assigneeIds, assignee.Username)
			}
			assigneesJSON, _ := json.Marshal(assigneeIds)
			task.Assignees = string(assigneesJSON)

			// Note: We intentionally do NOT extract PR links from description
			// PR links are extracted from task comments in the ExtractComments subtask
			// This is more reliable and follows the recommended workflow

			return []interface{}{task}, nil
		},
	})

	if err != nil {
		return err
	}

	return extractor.Execute()
}

func parseClickUpTime(timeStr string) (time.Time, error) {
	// ClickUp timestamps are in milliseconds
	timestampMs, err := strconv.ParseInt(timeStr, 10, 64)
	if err != nil {
		return time.Time{}, err
	}
	return time.Unix(timestampMs/1000, (timestampMs%1000)*int64(time.Millisecond)), nil
}
