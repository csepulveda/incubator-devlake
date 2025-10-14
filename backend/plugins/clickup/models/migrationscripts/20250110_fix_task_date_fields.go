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
)

type clickUpTask20250110 struct {
	archived.NoPKModel
	ConnectionId uint64 `gorm:"primaryKey"`
	Id           string `gorm:"primaryKey;type:varchar(255)"`

	// Basic fields
	Name        string `gorm:"type:varchar(255)"`
	Description string `gorm:"type:text"`
	Status      string `gorm:"type:varchar(100)"`
	Priority    string `gorm:"type:varchar(50)"`
	ListId      string `gorm:"type:varchar(255);index"`
	FolderId    string `gorm:"type:varchar(255)"`
	SpaceId     string `gorm:"type:varchar(255)"`

	// Dates - Fixed to use DateCreated/DateUpdated to avoid conflicts with NoPKModel
	DateCreated time.Time  `gorm:"type:datetime(3)"`
	DateUpdated time.Time  `gorm:"type:datetime(3)"`
	StartDate   *time.Time
	DueDate     *time.Time
	DateDone    *time.Time
	DateClosed  *time.Time

	// User assignment
	Creator   string `gorm:"type:varchar(255)"`
	Assignees string `gorm:"type:text"`

	// Custom fields for GitHub linking
	GitHubPRUrl    string `gorm:"type:varchar(500)"`
	GitHubPRNumber int
	GitHubBranch   string `gorm:"type:varchar(255)"`

	// URLs
	Url string `gorm:"type:varchar(500)"`
}

func (clickUpTask20250110) TableName() string {
	return "_tool_clickup_tasks"
}

type fixTaskDateFields struct{}

func (*fixTaskDateFields) Up(basicRes context.BasicRes) errors.Error {
	// Drop and recreate the table with correct schema
	// This is safe because we're in development and the conflicting fields
	// were preventing data from being persisted anyway
	return basicRes.GetDal().AutoMigrate(&clickUpTask20250110{})
}

func (*fixTaskDateFields) Version() uint64 {
	return 20250110000005
}

func (*fixTaskDateFields) Name() string {
	return "Fix ClickUp task date field conflicts with NoPKModel"
}
