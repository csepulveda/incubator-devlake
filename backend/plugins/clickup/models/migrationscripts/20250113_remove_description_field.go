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

var _ plugin.MigrationScript = (*removeDescriptionField)(nil)

type removeDescriptionField struct{}

func (*removeDescriptionField) Up(basicRes context.BasicRes) errors.Error {
	// Remove the description field from _tool_clickup_tasks table
	// Description data is not needed - we only use comments for PR linking
	db := basicRes.GetDal()

	err := db.Exec(`
		ALTER TABLE _tool_clickup_tasks
		DROP COLUMN description
	`)
	if err != nil {
		return err
	}

	return nil
}

func (*removeDescriptionField) Version() uint64 {
	return 20250113000002
}

func (*removeDescriptionField) Name() string {
	return "Remove description field from ClickUp tasks - only comments needed for PR linking"
}
