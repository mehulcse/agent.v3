package main

import (
	"context"
	"net/url"
	"strings"

	"github.com/hashicorp/go-hclog"
	"github.com/pinpt/agent/integrations/gitlab/api"
	"github.com/pinpt/agent/rpcdef"
	"github.com/pinpt/integration-sdk/agent"
)

func (s *Integration) OnboardExport(ctx context.Context, objectType rpcdef.OnboardExportType, config rpcdef.ExportConfig) (res rpcdef.OnboardExportResult, _ error) {
	err := s.initWithConfig(config)
	if err != nil {
		return res, err
	}
	switch objectType {
	case rpcdef.OnboardExportTypeRepos, rpcdef.OnboardExportTypeProjects:
		return s.onboardExportRepos(ctx, objectType)
	case rpcdef.OnboardExportTypeWorkConfig:
		return s.onboardWorkConfig(ctx, config.Integration.ID)
	default:
		res.Error = rpcdef.ErrOnboardExportNotSupported
		return
	}
}

func (s *Integration) onboardWorkConfig(ctx context.Context, intid string) (res rpcdef.OnboardExportResult, _ error) {

	ws := &agent.WorkStatusResponseWorkConfig{}
	ws.CustomerID = s.customerID
	ws.IntegrationID = intid
	ws.RefType = "gitlab"
	ws.Statuses = agent.WorkStatusResponseWorkConfigStatuses{
		OpenStatus:       []string{"open", "Open"},
		InProgressStatus: []string{"in progress", "In progress", "In Progress"},
		ClosedStatus:     []string{"closed", "Closed"},
	}
	ws.TopLevelIssue = agent.WorkStatusResponseWorkConfigTopLevelIssue{
		Name: "Issue",
		Type: "Issue",
	}

	res.Data = ws.ToMap()
	return
}

func (s *Integration) onboardExportRepos(ctx context.Context, objectType rpcdef.OnboardExportType) (res rpcdef.OnboardExportResult, _ error) {
	groups, err := api.GroupsAll(s.qc)
	if err != nil {
		return res, err
	}

	var records []map[string]interface{}

	for _, group := range groups {
		err := api.PaginateStartAt(s.logger, func(log hclog.Logger, paginationParams url.Values) (page api.PageInfo, _ error) {
			pi, repos, err := api.ReposOnboardPage(s.qc, group, paginationParams)
			if err != nil {
				return pi, err
			}
			for _, repo := range repos {

				if objectType == rpcdef.OnboardExportTypeRepos {
					records = append(records, repo.ToMap())
				} else {
					var identifier string
					if parts := strings.Split(repo.Name, "/"); len(parts) == 2 {
						identifier = parts[1]
					} else {
						identifier = repo.Name
					}
					records = append(records, (&agent.ProjectResponseProjects{
						Active:            repo.Active,
						Description:       &repo.Description,
						Identifier:        identifier,
						Error:             agent.ProjectResponseProjectsError(repo.Error),
						Name:              repo.Name,
						RefID:             repo.RefID,
						RefType:           repo.RefType,
						WebhookPermission: repo.WebhookPermission,
					}).ToMap())
				}
			}
			return pi, nil
		})
		if err != nil {
			return res, err
		}
	}

	res.Data = records

	return res, nil
}
