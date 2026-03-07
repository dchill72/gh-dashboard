package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type Client struct {
	host  string
	token string
	http  *http.Client
}

func NewClient(host, token string) *Client {
	return &Client{
		host:  host,
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) graphqlURL() string {
	if c.host == "github.com" {
		return "https://api.github.com/graphql"
	}
	return fmt.Sprintf("https://%s/api/graphql", c.host)
}

const prSearchQuery = `
query ReviewerPRs($searchQuery: String!, $cursor: String) {
  search(query: $searchQuery, type: ISSUE, first: 100, after: $cursor) {
    pageInfo {
      hasNextPage
      endCursor
    }
    nodes {
      ... on PullRequest {
        id
        number
        title
        url
        createdAt
        updatedAt
        isDraft
        body
        repository {
          nameWithOwner
        }
        author {
          login
        }
        commits {
          totalCount
        }
        changedFiles
        additions
        deletions
        reviewDecision
      }
    }
  }
}
`

type graphqlRequest struct {
	Query     string                 `json:"query"`
	Variables map[string]interface{} `json:"variables"`
}

type graphqlResponse struct {
	Data   *searchData    `json:"data"`
	Errors []graphqlError `json:"errors"`
}

type graphqlError struct {
	Message string `json:"message"`
}

type searchData struct {
	Search searchResult `json:"search"`
}

type searchResult struct {
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
	Nodes []prNode `json:"nodes"`
}

type prNode struct {
	ID         string    `json:"id"`
	Number     int       `json:"number"`
	Title      string    `json:"title"`
	URL        string    `json:"url"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
	IsDraft    bool      `json:"isDraft"`
	Body       string    `json:"body"`
	Repository struct {
		NameWithOwner string `json:"nameWithOwner"`
	} `json:"repository"`
	Author struct {
		Login string `json:"login"`
	} `json:"author"`
	Commits struct {
		TotalCount int `json:"totalCount"`
	} `json:"commits"`
	ChangedFiles   int    `json:"changedFiles"`
	Additions      int    `json:"additions"`
	Deletions      int    `json:"deletions"`
	ReviewDecision string `json:"reviewDecision"`
}

func (c *Client) FetchReviewerPRs(orgs []OrgQuery) ([]PR, error) {
	var all []PR
	for _, org := range orgs {
		prs, err := c.fetchForOrg(org)
		if err != nil {
			return nil, fmt.Errorf("fetching org %s: %w", org.Name, err)
		}
		all = append(all, prs...)
	}
	return all, nil
}

func (c *Client) fetchForOrg(org OrgQuery) ([]PR, error) {
	var searchQuery string
	if len(org.Repos) > 0 {
		for _, repo := range org.Repos {
			searchQuery += fmt.Sprintf(" repo:%s/%s", org.Name, repo)
		}
		searchQuery = "is:pr is:open review-requested:@me" + searchQuery
	} else {
		searchQuery = fmt.Sprintf("is:pr is:open review-requested:@me org:%s", org.Name)
	}

	var prs []PR
	var cursor *string

	for {
		vars := map[string]interface{}{
			"searchQuery": searchQuery,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		result, err := c.doGraphQL(vars)
		if err != nil {
			return nil, err
		}

		for _, node := range result.Nodes {
			prs = append(prs, PR{
				ID:             node.ID,
				Number:         node.Number,
				Title:          node.Title,
				URL:            node.URL,
				Repo:           node.Repository.NameWithOwner,
				Author:         node.Author.Login,
				CreatedAt:      node.CreatedAt,
				UpdatedAt:      node.UpdatedAt,
				Body:           node.Body,
				Commits:        node.Commits.TotalCount,
				ChangedFiles:   node.ChangedFiles,
				Additions:      node.Additions,
				Deletions:      node.Deletions,
				ReviewDecision: node.ReviewDecision,
				IsDraft:        node.IsDraft,
			})
		}

		if !result.PageInfo.HasNextPage {
			break
		}
		cur := result.PageInfo.EndCursor
		cursor = &cur
	}

	return prs, nil
}

func (c *Client) doGraphQL(variables map[string]interface{}) (*searchResult, error) {
	body, err := json.Marshal(graphqlRequest{
		Query:     prSearchQuery,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.graphqlURL(), bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	resp, err := c.http.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return nil, fmt.Errorf("authentication failed: check your GITHUB_TOKEN")
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	var gqlResp graphqlResponse
	if err := json.NewDecoder(resp.Body).Decode(&gqlResp); err != nil {
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if gqlResp.Data == nil {
		return nil, fmt.Errorf("no data in response")
	}

	return &gqlResp.Data.Search, nil
}
