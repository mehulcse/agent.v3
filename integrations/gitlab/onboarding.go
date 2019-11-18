package main

import (
	"context"
	"net/url"

	"github.com/hashicorp/go-hclog"
	"github.com/pinpt/agent.next/integrations/gitlab/api"
	"github.com/pinpt/agent.next/rpcdef"
	pjson "github.com/pinpt/go-common/json"
)

func (s *Integration) OnboardExport(ctx context.Context, objectType rpcdef.OnboardExportType, config rpcdef.ExportConfig) (res rpcdef.OnboardExportResult, _ error) {
	err := s.initWithConfig(config)
	if err != nil {
		return res, err
	}
	switch objectType {
	case rpcdef.OnboardExportTypeRepos:
		return s.onboardExportRepos(ctx)
	default:
		res.Error = rpcdef.ErrOnboardExportNotSupported
		return
	}
}

func (s *Integration) onboardExportRepos(ctx context.Context) (res rpcdef.OnboardExportResult, _ error) {
	groupNames, err := api.GroupsAll(s.qc)
	if err != nil {
		return res, err
	}

	var records []map[string]interface{}

	for _, groupName := range groupNames {
		api.PaginateStartAt(s.logger, func(log hclog.Logger, paginationParams url.Values) (page api.PageInfo, _ error) {
			pi, repos, err := api.ReposOnboardPage(s.qc, groupName, paginationParams)
			if err != nil {
				return pi, err
			}
			for _, repo := range repos {
				records = append(records, repo.ToMap())
			}
			return pi, nil
		})
	}

	res.Data = records

	s.logger.Info("", "REPOS", pjson.Stringify(records))

	return res, nil
}
