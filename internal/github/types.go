package github

import "time"

type PR struct {
	ID             string
	Number         int
	Title          string
	URL            string
	Repo           string
	Author         string
	CreatedAt      time.Time
	UpdatedAt      time.Time
	Body           string
	Commits        int
	ChangedFiles   int
	Additions      int
	Deletions      int
	ReviewDecision string // APPROVED | CHANGES_REQUESTED | REVIEW_REQUIRED | ""
	IsDraft        bool
}

type OrgQuery struct {
	Name  string
	Repos []string
}
