package jiracommonapi

import "github.com/pinpt/integration-sdk/work"

func Priorities(qc QueryContext) (res []work.IssuePriority, rerr error) {

	objectPath := "priority"

	var rawPriorities []struct {
		ID          string `json:"id"`
		Name        string `json:"name"`
		Description string `json:"description"`
		Color       string `json:"statusColor"`
		Icon        string `json:"iconUrl"`
	}

	err := qc.Req.Get(objectPath, nil, &rawPriorities)
	if err != nil {
		rerr = err
		return
	}

	// the result comes back in priority order from HIGH (0) to LOW (length-1)
	// so we iterate backwards to make the highest first and the lowest last

	for order := len(rawPriorities) - 1; order >= 0; order-- {
		priority := rawPriorities[order]
		res = append(res, work.IssuePriority{
			ID:          work.NewIssuePriorityID(qc.CustomerID, "jira", priority.ID),
			CustomerID:  qc.CustomerID,
			Name:        priority.Name,
			Description: &priority.Description,
			IconURL:     &priority.Icon,
			Color:       &priority.Color,
			Order:       int64(order),
			RefType:     "jira",
			RefID:       priority.ID,
		})
	}

	return
}
