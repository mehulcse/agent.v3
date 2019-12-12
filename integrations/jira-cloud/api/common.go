package api

import (
	"net/url"

	"github.com/hashicorp/go-hclog"
	"github.com/pinpt/agent.next/integrations/pkg/jiracommonapi"
)

type QueryContext struct {
	WebsiteURL string
	Logger     hclog.Logger
	CustomerID string
	Request    func(objPath string, params url.Values, res interface{}) error
	Request2   func(objPath string, params url.Values, res interface{}) (statusCode int, _ error)
}

type PageInfo struct {
	Total      int
	MaxResults int
	HasMore    bool
}

func (s *QueryContext) Common() jiracommonapi.QueryContext {
	res := jiracommonapi.QueryContext{}
	res.WebsiteURL = s.WebsiteURL
	res.CustomerID = s.CustomerID
	res.Logger = s.Logger
	res.ExportUser = nil
	res.Request = s.Request
	res.Request2 = s.Request2
	res.Validate()
	return res
}
