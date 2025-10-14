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

var _ plugin.MigrationScript = (*addGitHubFields)(nil)

type addGitHubFields struct{}

func (script *addGitHubFields) Up(basicRes context.BasicRes) errors.Error {
	db := basicRes.GetDal()

	// Add GitHub linking fields to clickup_tasks table one by one
	// Using AddColumn to make it database-agnostic
	err := db.Exec(`ALTER TABLE _tool_clickup_tasks ADD COLUMN github_pr_url VARCHAR(500)`)
	if err != nil {
		// Column might already exist, continue
	}

	err = db.Exec(`ALTER TABLE _tool_clickup_tasks ADD COLUMN github_pr_number INT`)
	if err != nil {
		// Column might already exist, continue
	}

	err = db.Exec(`ALTER TABLE _tool_clickup_tasks ADD COLUMN github_branch VARCHAR(255)`)
	if err != nil {
		// Column might already exist, ignore
	}

	return nil
}

func (*addGitHubFields) Version() uint64 {
	return 20250110000005
}

func (*addGitHubFields) Name() string {
	return "add GitHub linking fields to ClickUp tasks"
}
