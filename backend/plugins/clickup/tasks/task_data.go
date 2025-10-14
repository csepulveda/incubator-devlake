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
	"net/http"
	"strconv"
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/clickup/models"
)

type ClickUpOptions struct {
	ConnectionId uint64 `json:"connectionId"`
	ScopeId      string `json:"scopeId"` // ClickUp Space ID (scope) to collect tasks from
	// Since is handled automatically by StatefulApiCollector from blueprint's TimeAfter
}

type ClickUpTaskData struct {
	Options   *ClickUpOptions
	ApiClient *helper.ApiAsyncClient
	FolderIds []string // Folder IDs collected from the space
	ListIds   []string // List IDs collected from folders and space
}

func CreateApiClient(taskCtx plugin.TaskContext, connection *models.ClickUpConnection) (*helper.ApiAsyncClient, errors.Error) {
	apiClient, err := helper.NewApiClientFromConnection(taskCtx.GetContext(), taskCtx, connection)
	if err != nil {
		return nil, err
	}

	// Set up authentication
	apiClient.SetHeaders(map[string]string{
		"Authorization": connection.Token,
		"Content-Type":  "application/json",
	})

	// Set rate limit - ClickUp uses per-minute limits
	// Default: 5000 requests/hour (83/min) - aggressive but monitored for 100/min limit
	rateLimitPerHour := connection.RateLimitPerHour
	if rateLimitPerHour == 0 {
		rateLimitPerHour = 5000 // 83 requests per minute (83% of 100/min limit)
	}

	apiClient.SetAfterFunction(func(res *http.Response) errors.Error {
		if res.StatusCode == http.StatusUnauthorized {
			return errors.Unauthorized.New("ClickUp API authentication failed")
		}
		if res.StatusCode == http.StatusTooManyRequests {
			// Return a retryable error for rate limiting
			return errors.HttpStatus(http.StatusTooManyRequests).New("ClickUp API rate limit exceeded")
		}

		// Monitor rate limit headers and log when getting close
		// ClickUp uses: x-ratelimit-limit, x-ratelimit-remaining, x-ratelimit-reset
		rateLimitRemaining := res.Header.Get("x-ratelimit-remaining")
		rateLimitLimit := res.Header.Get("x-ratelimit-limit")
		if rateLimitRemaining != "" && rateLimitLimit != "" {
			remaining, err1 := strconv.Atoi(rateLimitRemaining)
			limit, err2 := strconv.Atoi(rateLimitLimit)
			if err1 == nil && err2 == nil {
				percentRemaining := float64(remaining) / float64(limit) * 100

				// Log warning when below 20% remaining
				if percentRemaining < 20 && percentRemaining > 0 {
					taskCtx.GetLogger().Warn(nil, "ClickUp rate limit low: %d/%d remaining (%.1f%%)", remaining, limit, percentRemaining)
				}

				// Log info every 25% consumed (at 75%, 50%, 25% remaining)
				if remaining == limit*3/4 || remaining == limit/2 || remaining == limit/4 {
					taskCtx.GetLogger().Info("ClickUp rate limit status: %d/%d remaining (%.1f%%)", remaining, limit, percentRemaining)
				}
			}
		}
		return nil
	})

	asyncApiClient, err := helper.CreateAsyncApiClient(
		taskCtx,
		apiClient,
		&helper.ApiRateLimitCalculator{
			UserRateLimitPerHour: rateLimitPerHour,
			MaxRetry:             5,
			Method:               http.MethodGet,
			// DynamicRateLimit reads ClickUp's rate limit headers to adjust dynamically
			DynamicRateLimit: func(res *http.Response) (int, time.Duration, errors.Error) {
				// ClickUp uses: x-ratelimit-limit, x-ratelimit-remaining, x-ratelimit-reset
				headerLimit := res.Header.Get("x-ratelimit-limit")
				if headerLimit == "" {
					// No rate limit header, use configured value
					return 0, 0, nil
				}

				limit, err := strconv.Atoi(headerLimit)
				if err != nil {
					return 0, 0, nil
				}

				// ClickUp rate limits are per minute, convert to per hour
				return limit * 60, 1 * time.Hour, nil
			},
		},
	)
	if err != nil {
		return nil, err
	}

	return asyncApiClient, nil
}
