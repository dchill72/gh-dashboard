package github

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"gh-dashboard/internal/logger"
)

type Client struct {
	host        string
	token       string
	http        *http.Client
	viewerLogin string // cached after first call to GetViewerLogin
}

func NewClient(host, token string) *Client {
	return &Client{
		host:  host,
		token: token,
		http:  &http.Client{Timeout: 30 * time.Second},
	}
}

const viewerQuery = `query { viewer { login } }`

// GetViewerLogin returns the login of the authenticated user, fetching and
// caching it on the first call.
func (c *Client) GetViewerLogin() (string, error) {
	if c.viewerLogin != "" {
		return c.viewerLogin, nil
	}

	body, err := json.Marshal(graphqlRequest{Query: viewerQuery})
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", c.graphqlURL(), bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode == 401 {
		return "", fmt.Errorf("authentication failed: check your GITHUB_TOKEN")
	}

	var result struct {
		Data struct {
			Viewer struct {
				Login string `json:"login"`
			} `json:"viewer"`
		} `json:"data"`
		Errors []graphqlError `json:"errors"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", err
	}
	if len(result.Errors) > 0 {
		return "", fmt.Errorf("GraphQL error: %s", result.Errors[0].Message)
	}

	c.viewerLogin = result.Data.Viewer.Login
	logger.L.Info("resolved viewer login", "login", c.viewerLogin)
	return c.viewerLogin, nil
}

func (c *Client) graphqlURL() string {
	if c.host == "github.com" {
		return "https://api.github.com/graphql"
	}
	return fmt.Sprintf("https://%s/api/graphql", c.host)
}

const prSearchQuery = `
query PRs($searchQuery: String!, $cursor: String) {
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
	login, err := c.GetViewerLogin()
	if err != nil {
		return nil, fmt.Errorf("resolving current user: %w", err)
	}

	var all []PR
	for _, org := range orgs {
		prs, err := c.fetchForOrg(org, login)
		if err != nil {
			return nil, fmt.Errorf("fetching org %s: %w", org.Name, err)
		}
		all = append(all, prs...)
	}
	logger.L.Info("fetch complete", "total_prs", len(all))
	return all, nil
}

// fetchForOrg runs both the reviewer and assignee queries for a single org,
// then merges and deduplicates the results by PR ID.
func (c *Client) fetchForOrg(org OrgQuery, login string) ([]PR, error) {
	scope := orgScope(org)

	reviewerQuery := fmt.Sprintf("is:pr is:open review-requested:%s %s", login, scope)
	assigneeQuery := fmt.Sprintf("is:pr is:open assignee:%s %s", login, scope)

	logger.L.Debug("running reviewer query", "query", reviewerQuery)
	reviewerPRs, err := c.fetchAllPages(reviewerQuery)
	if err != nil {
		return nil, fmt.Errorf("reviewer query: %w", err)
	}
	logger.L.Info("reviewer query done", "org", org.Name, "count", len(reviewerPRs))

	logger.L.Debug("running assignee query", "query", assigneeQuery)
	assigneePRs, err := c.fetchAllPages(assigneeQuery)
	if err != nil {
		return nil, fmt.Errorf("assignee query: %w", err)
	}
	logger.L.Info("assignee query done", "org", org.Name, "count", len(assigneePRs))

	merged := mergePRs(reviewerPRs, assigneePRs)
	logger.L.Info("merged results", "org", org.Name, "merged_count", len(merged))
	return merged, nil
}

// orgScope builds the org/repo portion of the search query.
func orgScope(org OrgQuery) string {
	if len(org.Repos) == 0 {
		return fmt.Sprintf("org:%s", org.Name)
	}
	var scope string
	for _, repo := range org.Repos {
		scope += fmt.Sprintf(" repo:%s/%s", org.Name, repo)
	}
	return scope
}

// mergePRs combines reviewer and assignee slices, deduplicating by ID and
// accumulating Roles so a PR matched by both carries ["reviewer", "assignee"].
func mergePRs(reviewerPRs, assigneePRs []PR) []PR {
	seen := make(map[string]*PR, len(reviewerPRs)+len(assigneePRs))

	for i := range reviewerPRs {
		pr := reviewerPRs[i]
		pr.Roles = []string{"reviewer"}
		seen[pr.ID] = &pr
	}

	for i := range assigneePRs {
		pr := assigneePRs[i]
		if existing, ok := seen[pr.ID]; ok {
			existing.Roles = append(existing.Roles, "assignee")
		} else {
			pr.Roles = []string{"assignee"}
			seen[pr.ID] = &pr
		}
	}

	// Preserve order: reviewer results first, then assignee-only additions.
	merged := make([]PR, 0, len(seen))
	added := make(map[string]bool, len(seen))
	for _, pr := range reviewerPRs {
		if !added[pr.ID] {
			merged = append(merged, *seen[pr.ID])
			added[pr.ID] = true
		}
	}
	for _, pr := range assigneePRs {
		if !added[pr.ID] {
			merged = append(merged, *seen[pr.ID])
			added[pr.ID] = true
		}
	}
	return merged
}

// fetchAllPages paginates through all results for a search query.
func (c *Client) fetchAllPages(searchQuery string) ([]PR, error) {
	var prs []PR
	var cursor *string
	page := 0

	for {
		page++
		vars := map[string]interface{}{
			"searchQuery": searchQuery,
		}
		if cursor != nil {
			vars["cursor"] = *cursor
		}

		result, err := c.doGraphQL(vars)
		if err != nil {
			logger.L.Error("graphql request failed", "query", searchQuery, "page", page, "err", err)
			return nil, err
		}
		logger.L.Debug("graphql page received",
			"query", searchQuery,
			"page", page,
			"nodes", len(result.Nodes),
			"has_next_page", result.PageInfo.HasNextPage,
		)

		for _, node := range result.Nodes {
			logger.L.Debug("pr node", "id", node.ID, "number", node.Number, "title", node.Title, "repo", node.Repository.NameWithOwner)
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
	reqBody, err := json.Marshal(graphqlRequest{
		Query:     prSearchQuery,
		Variables: variables,
	})
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("POST", c.graphqlURL(), bytes.NewReader(reqBody))
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
		body, _ := io.ReadAll(resp.Body)
		logger.L.Error("unexpected http status", "status", resp.Status, "body", string(body))
		return nil, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	// Tee the response body so we can log the raw JSON and still decode it.
	var rawBuf bytes.Buffer
	tee := io.TeeReader(resp.Body, &rawBuf)

	var gqlResp graphqlResponse
	if err := json.NewDecoder(tee).Decode(&gqlResp); err != nil {
		logger.L.Error("failed to decode response", "err", err, "raw", rawBuf.String())
		return nil, err
	}

	if len(gqlResp.Errors) > 0 {
		logger.L.Error("graphql errors", "errors", gqlResp.Errors)
		return nil, fmt.Errorf("GraphQL error: %s", gqlResp.Errors[0].Message)
	}

	if gqlResp.Data == nil {
		logger.L.Error("no data in response", "raw", rawBuf.String())
		return nil, fmt.Errorf("no data in response")
	}

	return &gqlResp.Data.Search, nil
}
