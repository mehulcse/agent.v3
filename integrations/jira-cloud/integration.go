package main

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/pkg/date"
	"github.com/pinpt/agent.next/pkg/objsender"

	"github.com/hashicorp/go-hclog"
	"github.com/pinpt/agent.next/integrations/jira-cloud/api"
	"github.com/pinpt/agent.next/integrations/pkg/ibase"
	"github.com/pinpt/agent.next/rpcdef"
	"github.com/pinpt/go-datamodel/work"
)

func main() {
	ibase.MainFunc(func(logger hclog.Logger) rpcdef.Integration {
		return NewIntegration(logger)
	})
}

type Integration struct {
	logger hclog.Logger
	agent  rpcdef.Agent
	config Config
	//selfHosted bool
	qc api.QueryContext
}

func NewIntegration(logger hclog.Logger) *Integration {
	s := &Integration{}
	s.logger = logger
	return s
}

func (s *Integration) Init(agent rpcdef.Agent) error {
	s.agent = agent
	return nil
}

type Config struct {
	URL      string
	Username string
	Password string
}

func (s *Integration) setIntegrationConfig(data map[string]interface{}) error {
	rerr := func(msg string, args ...interface{}) error {
		return fmt.Errorf("config validation error: "+msg, args...)
	}
	conf := Config{}

	conf.URL, _ = data["url"].(string)
	if conf.URL == "" {
		return rerr("url is missing")
	}

	conf.Username, _ = data["username"].(string)
	if conf.Username == "" {
		return rerr("username is missing")
	}

	conf.Password, _ = data["password"].(string)
	if conf.Password == "" {
		return rerr("password is missing")
	}

	s.config = conf
	return nil
}

func (s *Integration) Export(ctx context.Context, config rpcdef.ExportConfig) (res rpcdef.ExportResult, _ error) {
	err := s.setIntegrationConfig(config.Integration)
	if err != nil {
		return res, err
	}

	/*
		{
			u, err := url.Parse(s.config.URL)
			if err != nil {
				return res, err
			}
			if !strings.HasSuffix(u.Hostname(), ".atlassian.net") {
				s.selfHosted = true
			}
		}*/

	s.qc.CustomerID = config.Pinpoint.CustomerID
	s.qc.Logger = s.logger
	s.qc.BaseURL = s.config.URL

	{
		opts := api.RequesterOpts{}
		opts.Logger = s.logger
		opts.APIURL = s.config.URL
		opts.Username = s.config.Username
		opts.Password = s.config.Password
		//opts.SelfHosted = s.selfHosted
		requester := api.NewRequester(opts)

		s.qc.Request = requester.Request
	}

	users, err := NewUsers(s)
	if err != nil {
		return res, err
	}
	defer users.Done()
	s.qc.ExportUser = users.ExportUser

	fields, err := s.fields()
	if err != nil {
		return res, err
	}

	fieldByKey := map[string]*work.CustomField{}
	for _, f := range fields {
		fieldByKey[f.Key] = f
	}

	projects, err := s.projects()
	if err != nil {
		return res, err
	}

	err = s.issuesAndChangelogs(projects, fieldByKey)
	if err != nil {
		return res, err
	}

	return res, nil
}

type Project = api.Project

func (s *Integration) projects() (all []Project, _ error) {
	sender := objsender.NewNotIncremental(s.agent, "work.project")
	defer sender.Done()

	return all, api.PaginateStartAt(func(paginationParams url.Values) (hasMore bool, pageSize int, _ error) {
		pi, res, err := api.ProjectsPage(s.qc, paginationParams)
		if err != nil {
			return false, 0, err
		}
		for _, obj := range res {
			p := Project{}
			p.JiraID = obj.RefID
			p.Key = obj.Identifier
			all = append(all, p)
		}
		var res2 []objsender.Model
		for _, obj := range res {
			res2 = append(res2, obj)
		}
		err = sender.Send(res2)
		if err != nil {
			return false, 0, err
		}
		return pi.HasMore, pi.MaxResults, nil
	})
}

func (s *Integration) issuesAndChangelogs(projects []Project, fieldByKey map[string]*work.CustomField) error {
	senderIssues, err := objsender.NewIncrementalDateBased(s.agent, "work.issue")
	if err != nil {
		return err
	}
	defer senderIssues.Done()

	senderChangelogs := objsender.NewNotIncremental(s.agent, "work.changelog")
	defer senderChangelogs.Done()

	startedSprintExport := time.Now()
	sprints := NewSprints()

	for _, p := range projects {
		err := s.issuesAndChangelogsForProject(p, fieldByKey, senderIssues, senderChangelogs, sprints)
		if err != nil {
			return err
		}
	}

	senderSprints := objsender.NewNotIncremental(s.agent, "work.sprints")
	defer senderSprints.Done()

	var sprintModels []objsender.Model
	for _, data := range sprints.data {
		item := &work.Sprint{}
		item.CustomerID = s.qc.CustomerID
		item.RefType = "jira"
		item.RefID = strconv.Itoa(data.ID)

		// TODO: datamodel is missing goal?
		//item.Goal = data.Goal

		item.Name = data.Name

		startDate, err := api.ParseTime(data.StartDate)
		if err != nil {
			return fmt.Errorf("could not parse startdate for sprint: %v err: %v", data.Name, err)
		}
		date.ConvertToModel(startDate, &item.Started)

		endDate, err := api.ParseTime(data.EndDate)
		if err != nil {
			return fmt.Errorf("could not parse enddata for sprint: %v err: %v", data.Name, err)
		}
		date.ConvertToModel(endDate, &item.Ended)

		completeDate, err := api.ParseTime(data.CompleteDate)
		if err != nil {
			return fmt.Errorf("could not parse completed for sprint: %v err: %v", data.Name, err)
		}
		date.ConvertToModel(completeDate, &item.Completed)

		switch data.State {
		case "CLOSED":
			item.Status = work.SprintStatusClosed
		case "ACTIVE":
			item.Status = work.SprintStatusActive
		case "FUTURE":
			item.Status = work.SprintStatusFuture
		default:
			return fmt.Errorf("invalid status for sprint: %v", data.State)
		}

		date.ConvertToModel(startedSprintExport, &item.Fetched)

		sprintModels = append(sprintModels, item)
	}
	return senderSprints.Send(sprintModels)

	return nil
}

func (s *Integration) issuesAndChangelogsForProject(project Project, fieldByKey map[string]*work.CustomField, senderIssues *objsender.IncrementalDateBased, senderChangelogs *objsender.NotIncremental, sprints *Sprints) error {

	err := api.PaginateStartAt(func(paginationParams url.Values) (hasMore bool, pageSize int, _ error) {
		pi, resIssues, resChangelogs, err := api.IssuesAndChangelogsPage(s.qc, project, fieldByKey, senderIssues.LastProcessed, paginationParams)
		if err != nil {
			return false, 0, err
		}

		for _, issue := range resIssues {
			for _, f := range issue.CustomFields {
				if f.Name == "Sprint" {
					if f.Value == "" {
						continue
					}
					err := sprints.processIssueSprint(issue.RefID, f.Value)
					if err != nil {
						return false, 0, err
					}

					break
				}
			}
		}

		var resIssues2 []objsender.Model
		for _, obj := range resIssues {
			resIssues2 = append(resIssues2, obj)
		}
		err = senderIssues.Send(resIssues2)
		if err != nil {
			return false, 0, err
		}

		var resChangelogs2 []objsender.Model
		for _, obj := range resChangelogs {
			resChangelogs2 = append(resChangelogs2, obj)
		}
		err = senderChangelogs.Send(resChangelogs2)
		if err != nil {
			return false, 0, err
		}

		return pi.HasMore, pi.MaxResults, nil
	})
	if err != nil {
		return err
	}

	return nil
}

func (s *Integration) fields() ([]*work.CustomField, error) {
	sender := objsender.NewNotIncremental(s.agent, "work.custom_field")
	defer sender.Done()

	res, err := api.FieldsAll(s.qc)
	if err != nil {
		return nil, err
	}
	var res2 []objsender.Model
	for _, item := range res {
		res2 = append(res2, item)
	}
	return res, sender.Send(res2)
}
