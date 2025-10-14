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
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/errors"
	"github.com/apache/incubator-devlake/core/plugin"
)

var _ plugin.MigrationScript = (*addFolderIdsToScopeConfig)(nil)

type addFolderIdsToScopeConfig struct{}

func (script *addFolderIdsToScopeConfig) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Add folder_ids field to scope_config table
	err := db.Exec(`ALTER TABLE _tool_clickup_scope_configs ADD COLUMN folder_ids JSON`)
	if err != nil {
		// Column might already exist, ignore error
	}

	return nil
}

func (*addFolderIdsToScopeConfig) Version() uint64 {
	return 20250110000006
}

func (*addFolderIdsToScopeConfig) Name() string {
	return "add folder_ids to ClickUp scope config for folder selection"
}
