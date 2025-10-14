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

package models

import (
	"time"

	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/common"
	"github.com/apache/incubator-devlake/core/plugin"
	helper "github.com/apache/incubator-devlake/helpers/pluginhelper/api"
)

// ClickUpTask represents a task from ClickUp
type ClickUpTask struct {
	common.NoPKModel
	ConnectionId uint64 `gorm:"primaryKey"`
	Id           string `gorm:"primaryKey;type:varchar(255)"`

	// Basic fields
	Name        string     `gorm:"type:varchar(255)"` // Truncated to 255 chars in extractor
	// Description field removed - we only need comments for PR linking, not full descriptions
	Status      string     `gorm:"type:varchar(100)"`
	Priority    string     `gorm:"type:varchar(50)"`
	ListId      string     `gorm:"type:varchar(255);index"`
	FolderId    string     `gorm:"type:varchar(255)"`
	SpaceId     string     `gorm:"type:varchar(255)"`

	// Dates - Use DateCreated/DateUpdated to avoid conflicts with NoPKModel's CreatedAt/UpdatedAt
	DateCreated  time.Time  `gorm:"type:datetime(3)" json:"date_created"`
	DateUpdated  time.Time  `gorm:"type:datetime(3)" json:"date_updated"`
	StartDate    *time.Time `json:"start_date"`
	DueDate      *time.Time `json:"due_date"`
	DateDone     *time.Time `json:"date_done"`
	DateClosed   *time.Time `json:"date_closed"`

	// User assignment
	Creator   string `gorm:"type:varchar(255)"`
	Assignees string `gorm:"type:text"` // JSON array of assignee IDs

	// Custom fields for GitHub linking
	GitHubPRUrl    string `gorm:"type:varchar(500)"`
	GitHubPRNumber int
	GitHubBranch   string `gorm:"type:varchar(255)"`

	// URLs
	Url string `gorm:"type:varchar(500)"`
}

func (ClickUpTask) TableName() string {
	return "_tool_clickup_tasks"
}

// ClickUpConnection stores connection info
type ClickUpConnection struct {
	helper.BaseConnection `mapstructure:",squash"`
	RestConnection        `mapstructure:",squash"`
	Token                 string `mapstructure:"token" validate:"required" json:"token"`
}

type RestConnection struct {
	Endpoint         string `mapstructure:"endpoint" validate:"required" json:"endpoint"`
	Proxy            string `mapstructure:"proxy" json:"proxy"`
	RateLimitPerHour int    `mapstructure:"rateLimitPerHour" json:"rateLimitPerHour"`
}

func (c ClickUpConnection) GetEndpoint() string {
	return c.Endpoint
}

func (c ClickUpConnection) GetProxy() string {
	return c.Proxy
}

func (c ClickUpConnection) GetRateLimitPerHour() int {
	return c.RateLimitPerHour
}

func (ClickUpConnection) TableName() string {
	return "_tool_clickup_connections"
}

// PrepareApiClient configures the API client with ClickUp-specific authentication
func (c *ClickUpConnection) PrepareApiClient(apiClient plugin.ApiClient) errors.Error {
	// ClickUp uses a simple token authentication with format: Authorization: {token}
	apiClient.SetHeaders(map[string]string{
		"Authorization": c.Token,
		"Content-Type":  "application/json",
	})
	return nil
}

// ClickUpTaskDeploymentRelationship links ClickUp tasks to deployments
type ClickUpTaskDeploymentRelationship struct {
	TaskId             string    `gorm:"primaryKey;type:varchar(255)"`
	DeploymentId       string    `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId       uint64    `gorm:"primaryKey"`

	// Timestamps for metrics
	TaskCreatedAt      time.Time
	FirstCommitAt      *time.Time
	PRCreatedAt        *time.Time
	PRMergedAt         *time.Time
	DeploymentAt       *time.Time

	// Calculated metrics (in minutes)
	PlanningTime       *int // TaskCreated → FirstCommit
	CodeTime           *int // FirstCommit → PRMerged
	DeployTime         *int // PRMerged → Deployment
	TotalCycleTime     *int // TaskCreated → Deployment
}

func (ClickUpTaskDeploymentRelationship) TableName() string {
	return "_tool_clickup_task_deployments"
}
