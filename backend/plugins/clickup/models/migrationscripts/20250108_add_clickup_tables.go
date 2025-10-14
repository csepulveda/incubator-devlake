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

package migrationscripts

import (
	"time"

	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/models/migrationscripts/archived"
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addClickUpTables)(nil)

type clickUpConnection20250108 struct {
	archived.BaseConnection `mapstructure:",squash"`
	archived.RestConnection `mapstructure:",squash"`
	Token                   string `gorm:"type:varchar(255)" mapstructure:"token" validate:"required" json:"token"`
}

func (clickUpConnection20250108) TableName() string {
	return "_tool_clickup_connections"
}

type clickUpTask20250108 struct {
	archived.NoPKModel
	ConnectionId uint64 `gorm:"primaryKey"`
	Id           string `gorm:"primaryKey;type:varchar(255)"`

	// Basic fields
	Name        string     `gorm:"type:varchar(255)"`
	Description string     `gorm:"type:text"`
	Status      string     `gorm:"type:varchar(100)"`
	Priority    string     `gorm:"type:varchar(50)"`
	ListId      string     `gorm:"type:varchar(255);index"`
	FolderId    string     `gorm:"type:varchar(255)"`
	SpaceId     string     `gorm:"type:varchar(255)"`

	// Dates
	CreatedAt  time.Time  `json:"date_created"`
	UpdatedAt  time.Time  `json:"date_updated"`
	StartDate  *time.Time `json:"start_date"`
	DueDate    *time.Time `json:"due_date"`
	DateDone   *time.Time `json:"date_done"`
	DateClosed *time.Time `json:"date_closed"`

	// User assignment
	Creator   string `gorm:"type:varchar(255)"`
	Assignees string `gorm:"type:text"` // JSON array

	// GitHub linking
	GitHubPRUrl    string `gorm:"type:varchar(500)"`
	GitHubPRNumber int
	GitHubBranch   string `gorm:"type:varchar(255)"`

	// URLs
	Url string `gorm:"type:varchar(500)"`
}

func (clickUpTask20250108) TableName() string {
	return "_tool_clickup_tasks"
}

type clickUpTaskDeploymentRelationship20250108 struct {
	TaskId       string `gorm:"primaryKey;type:varchar(255)"`
	DeploymentId string `gorm:"primaryKey;type:varchar(255)"`
	ConnectionId uint64 `gorm:"primaryKey"`

	// Timestamps
	TaskCreatedAt time.Time
	FirstCommitAt *time.Time
	PRCreatedAt   *time.Time
	PRMergedAt    *time.Time
	DeploymentAt  *time.Time

	// Metrics (in minutes)
	PlanningTime   *int
	CodeTime       *int
	DeployTime     *int
	TotalCycleTime *int
}

func (clickUpTaskDeploymentRelationship20250108) TableName() string {
	return "_tool_clickup_task_deployments"
}

type clickUpScope20250108 struct {
	archived.ScopeConfig
	Id   string `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Name string `gorm:"type:varchar(255)" json:"name"`
}

func (clickUpScope20250108) TableName() string {
	return "_tool_clickup_scopes"
}

type clickUpScopeConfig20250108 struct {
	archived.ScopeConfig
}

func (clickUpScopeConfig20250108) TableName() string {
	return "_tool_clickup_scope_configs"
}

type addClickUpTables struct{}

func (script *addClickUpTables) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()
	err := db.AutoMigrate(&clickUpConnection20250108{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&clickUpTask20250108{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&clickUpTaskDeploymentRelationship20250108{})
	if err != nil {
		return err
	}
	err = db.AutoMigrate(&clickUpScope20250108{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(&clickUpScopeConfig20250108{})
}

func (*addClickUpTables) Version() uint64 {
	return 20250108000001
}

func (*addClickUpTables) Name() string {
	return "add ClickUp tables for business cycle time tracking"
}
