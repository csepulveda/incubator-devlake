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
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*fixClickUpScopeTables)(nil)

type clickUpScope20250110v2 struct {
	ConnectionId  uint64     `gorm:"primaryKey"`
	Id            string     `gorm:"primaryKey;type:varchar(255)" json:"id"`
	Name          string     `gorm:"type:varchar(255)" json:"name"`
	ScopeConfigId *uint64    `gorm:"column:scope_config_id"`
	CreatedAt     *time.Time `json:"createdAt"`
	UpdatedAt     *time.Time `json:"updatedAt"`
	// RawDataOrigin fields
	RawDataParams string `gorm:"column:_raw_data_params;type:varchar(255);index"`
	RawDataTable  string `gorm:"column:_raw_data_table;type:varchar(255)"`
	RawDataId     uint64 `gorm:"column:_raw_data_id"`
	RawDataRemark string `gorm:"column:_raw_data_remark"`
}

func (clickUpScope20250110v2) TableName() string {
	return "_tool_clickup_scopes"
}

type clickUpScopeConfig20250110v2 struct {
	Id           uint64     `gorm:"primaryKey" json:"id"`
	ConnectionId uint64     `json:"connectionId" gorm:"index"`
	Name         string     `gorm:"type:varchar(255);index" json:"name"`
	Entities     []string   `gorm:"type:json;serializer:json" json:"entities"`
	CreatedAt    *time.Time `json:"createdAt"`
	UpdatedAt    *time.Time `json:"updatedAt"`
}

func (clickUpScopeConfig20250110v2) TableName() string {
	return "_tool_clickup_scope_configs"
}

type fixClickUpScopeTables struct{}

func (script *fixClickUpScopeTables) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Drop existing tables (they were created with wrong structure)
	err := db.DropTables(&clickUpScope20250110v2{})
	if err != nil {
		return err
	}
	err = db.DropTables(&clickUpScopeConfig20250110v2{})
	if err != nil {
		return err
	}

	// Recreate tables with correct structure
	err = db.AutoMigrate(&clickUpScope20250110v2{})
	if err != nil {
		return err
	}
	return db.AutoMigrate(&clickUpScopeConfig20250110v2{})
}

func (*fixClickUpScopeTables) Version() uint64 {
	return 20250110000004
}

func (*fixClickUpScopeTables) Name() string {
	return "fix ClickUp scope tables structure"
}
