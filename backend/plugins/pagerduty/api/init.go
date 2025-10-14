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

package api

import (
	"github.com/apache/incubator-devlake/core/context"
	"github.com/apache/incubator-devlake/core/dal"
	"github.com/apache/incubator-devlake/core/plugin"
	"github.com/apache/incubator-devlake/helpers/pluginhelper/api"
	"github.com/apache/incubator-devlake/plugins/pagerduty/models"
	"github.com/go-playground/validator/v10"
)

var vld *validator.Validate
var basicRes context.BasicRes

var dsHelper *api.DsHelper[models.PagerDutyConnection, models.Service, models.PagerdutyScopeConfig]
var raProxy *api.DsRemoteApiProxyHelper[models.PagerDutyConnection]
var raScopeList *api.DsRemoteApiScopeListHelper[models.PagerDutyConnection, models.Service, PagerdutyRemotePagination]

var raScopeSearch *api.DsRemoteApiScopeSearchHelper[models.PagerDutyConnection, models.Service]

func Init(br context.BasicRes, p plugin.PluginMeta) {
	vld = validator.New()
	basicRes = br
	dsHelper = api.NewDataSourceHelper[
		models.PagerDutyConnection, models.Service, models.PagerdutyScopeConfig,
	](
		br,
		p.Name(),
		[]string{"name"},
		func(c models.PagerDutyConnection) models.PagerDutyConnection {
			return c.Sanitize()
		},
		ensureScopeConfigId, // Add scope config assignment before save
		nil,
	)
	raProxy = api.NewDsRemoteApiProxyHelper[models.PagerDutyConnection](dsHelper.ConnApi.ModelApiHelper)
	raScopeList = api.NewDsRemoteApiScopeListHelper[models.PagerDutyConnection, models.Service, PagerdutyRemotePagination](raProxy, listPagerdutyRemoteScopes)
	raScopeSearch = api.NewDsRemoteApiScopeSearchHelper[models.PagerDutyConnection, models.Service](raProxy, searchPagerdutyRemoteScopes)
}

// ensureScopeConfigId automatically assigns a scope_config_id if not set
func ensureScopeConfigId(service models.Service) models.Service {
	// If already has a scope_config_id, don't change it
	if service.ScopeConfigId > 0 {
		return service
	}

	// Try to find a scope config for this connection
	var scopeConfig models.PagerdutyScopeConfig
	err := basicRes.GetDal().First(
		&scopeConfig,
		dal.Where("connection_id = ? AND id > 0", service.ConnectionId),
		dal.Orderby("id ASC"),
	)

	if err != nil {
		// If no scope config found, that's OK - scope_config_id stays 0
		if basicRes.GetDal().IsErrorNotFound(err) {
			basicRes.GetLogger().Debug("No scope config found for connection %d, service %s will have scope_config_id=0", service.ConnectionId, service.Id)
		} else {
			basicRes.GetLogger().Warn(err, "Error finding scope config for connection %d, service %s", service.ConnectionId, service.Id)
		}
		return service
	}

	// Assign the found scope_config_id
	service.ScopeConfigId = scopeConfig.ID
	basicRes.GetLogger().Info("Auto-assigned scope_config_id=%d to service %s (connection %d)", scopeConfig.ID, service.Id, service.ConnectionId)
	return service
}
