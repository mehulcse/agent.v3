package api

import (
	"fmt"
	"net/url"
	"strconv"
	"time"

	"github.com/pinpt/agent.next/pkg/date"
	"github.com/pinpt/agent.next/pkg/ids"
	"github.com/pinpt/agent.next/pkg/objsender"
	"github.com/pinpt/go-common/hash"
	pstrings "github.com/pinpt/go-common/strings"
	"github.com/pinpt/go-datamodel/sourcecode"
)

func PullRequestPage(
	qc QueryContext,
	repoID string,
	repoName string,
	params url.Values,
	stopOnUpdatedAt time.Time,
	reviewsSender *objsender.NotIncremental) (pi PageInfo, res []sourcecode.PullRequest, err error) {

	qc.Logger.Debug("repo pull requests", "repo", repoName)

	objectPath := pstrings.JoinURL("repositories", repoName, "pullrequests")
	params.Add("state", "MERGED")
	params.Add("state", "SUPERSEDED")
	params.Add("state", "OPEN")
	params.Add("state", "DECLINED")
	params.Set("sort", "-updated_on")
	// Greater than 50 throws "Invalid pagelen"
	params.Set("pagelen", "50")

	var rprs []struct {
		ID     int64 `json:"id"`
		Source struct {
			Branch struct {
				Name string `json:"name"`
			} `json:"branch"`
		} `json:"source"`
		Title       string `json:"title"`
		Description string `json:"description"`
		Links       struct {
			HTML struct {
				Href string `json:"href"`
			} `json:"html"`
		} `json:"links"`
		CreatedOn time.Time `json:"created_on"`
		UpdatedOn time.Time `json:"updated_on"`
		State     string    `json:"state"`
		ClosedBy  struct {
			AccountID string `json:"account_id"`
		} `json:"closed_by"`
		MergeCommit struct {
			Hash string `json:"hash"`
		} `json:"merge_commit"`
		Author struct {
			AccountID string `json:"account_id"`
		} `json:"author"`
		Participants []struct {
			Role           string    `json:"role"`
			Approved       bool      `json:"approved"`
			ParticipatedOn time.Time `json:"participated_on"`
			User           struct {
				AccountID string `json:"account_id"`
			} `json:"user"`
		} `json:"participants"`
	}

	pi, err = qc.Request(objectPath, params, true, &rprs)
	if err != nil {
		return
	}

	for _, rpr := range rprs {
		if rpr.UpdatedOn.Before(stopOnUpdatedAt) {
			return pi, res, nil
		}
		pr := sourcecode.PullRequest{}
		pr.CustomerID = qc.CustomerID
		pr.RefType = qc.RefType
		pr.RefID = fmt.Sprint(rpr.ID)
		pr.RepoID = ids.RepoID(repoID, qc)
		pr.BranchName = rpr.Source.Branch.Name
		pr.Title = rpr.Title
		pr.Description = rpr.Description
		pr.URL = rpr.Links.HTML.Href
		date.ConvertToModel(rpr.CreatedOn, &pr.CreatedDate)
		date.ConvertToModel(rpr.UpdatedOn, &pr.MergedDate)
		date.ConvertToModel(rpr.UpdatedOn, &pr.ClosedDate)
		date.ConvertToModel(rpr.UpdatedOn, &pr.UpdatedDate)
		switch rpr.State {
		case "OPEN":
			pr.Status = sourcecode.PullRequestStatusOpen
		case "DECLINED":
			pr.Status = sourcecode.PullRequestStatusClosed
			pr.ClosedByRefID = rpr.ClosedBy.AccountID
		case "MERGED":
			pr.MergeSha = rpr.MergeCommit.Hash
			pr.MergeCommitID = ids.CodeCommit(qc.CustomerID, qc.RefType, pr.RepoID, rpr.MergeCommit.Hash)
			pr.MergedByRefID = rpr.ClosedBy.AccountID
			pr.Status = sourcecode.PullRequestStatusMerged
		default:
			qc.Logger.Error("PR has an unknown state", "state", rpr.State, "ref_id", pr.RefID)
		}
		pr.CreatedByRefID = rpr.Author.AccountID

		res = append(res, pr)

		for _, participant := range rpr.Participants {
			if participant.Role == "REVIEWER" {
				review := sourcecode.PullRequestReview{}

				review.CustomerID = qc.CustomerID
				review.PullRequestID = strconv.FormatInt(rpr.ID, 10)
				review.RefID = hash.Values(pr.RefID, participant.User.AccountID)
				review.RefType = qc.RefType
				review.RepoID = ids.RepoID(repoID, qc)
				review.UpdatedAt = participant.ParticipatedOn.Unix()

				if participant.Approved {
					review.State = sourcecode.PullRequestReviewStateApproved
				} else {
					review.State = sourcecode.PullRequestReviewStatePending
				}

				review.UserRefID = participant.User.AccountID

				reviewsSender.Send(&review)
			}
		}

	}

	return
}
