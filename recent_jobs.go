package midjourney

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"
)

const FromDateFormat = "2006-01-02 15:04:05.999999"

var ErrUserIDRequired = fmt.Errorf("%w: user id required", Err)

type Order string

const (
	OrderHot       Order = "hot"
	OrderNew       Order = "new"
	OrderOldest    Order = "oldest"
	OrderTopToday  Order = "top-today"
	OrderTopWeekly Order = "top-weekly"
	OrderTopMonth  Order = "top-month"
	OrderTopAll    Order = "top-all"
	OrderLikedTime Order = "liked_timestamp"
)

type RecentJobsQuery struct {
	Amount      int
	JobType     JobType
	OrderBy     Order
	JobStatus   JobStatus
	UserID      string
	UserIDLiked string
	FromDate    time.Time
	Page        int
	Prompt      string
	Personal    bool
	Dedupe      bool
	RefreshAPI  int
}

func (rjq *RecentJobsQuery) Values() url.Values {
	v := url.Values{}
	if rjq.Amount != 0 {
		v.Set("amount", strconv.Itoa(rjq.Amount))
	}
	if rjq.JobType != "" {
		v.Set("jobType", string(rjq.JobType))
	}
	if rjq.OrderBy != "" {
		v.Set("orderBy", string(rjq.OrderBy))
	}
	if rjq.JobStatus != "" {
		v.Set("jobStatus", string(rjq.JobStatus))
	}
	if rjq.UserID != "" {
		v.Set("userId", rjq.UserID)
	}
	if rjq.UserIDLiked != "" {
		v.Set("userIdLiked", rjq.UserIDLiked)
	}
	if !rjq.FromDate.IsZero() {
		v.Set("fromDate", rjq.FromDate.Format(FromDateFormat))
	}
	if rjq.Page != 0 {
		v.Set("page", strconv.Itoa(rjq.Page))
	}
	if rjq.Prompt != "" {
		v.Set("prompt", rjq.Prompt)
	}
	if rjq.Personal {
		v.Set("personal", "true")
	}
	if rjq.Dedupe {
		v.Set("dedupe", "true")
	}
	v.Set("refreshApi", strconv.Itoa(rjq.RefreshAPI))

	return v
}

func (rjq *RecentJobsQuery) NextPage() *RecentJobsQuery {
	q := *rjq
	if q.OrderBy == OrderNew && q.FromDate.IsZero() {
		q.FromDate = time.Now().UTC()
	}
	if q.Page == 0 {
		q.Page = 1
	}
	q.Page = rjq.Page + 1

	return &q
}

type RecentJobs struct {
	Query RecentJobsQuery
	Jobs  []*Job
	Page  int
}

func (c *Client) RecentJobs(
	ctx context.Context,
	q *RecentJobsQuery,
) (*RecentJobs, error) {
	u := &url.URL{
		Path:     "app/recent-jobs/",
		RawQuery: q.Values().Encode(),
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()

	resp, err := c.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%w: %s", ErrResponseStatus, resp.Status)
	}

	rj := &RecentJobs{
		Query: *q,
		Jobs:  []*Job{},
		Page:  q.Page,
	}

	err = json.NewDecoder(resp.Body).Decode(&rj.Jobs)
	if err != nil {
		return nil, err
	}

	if rj.Query.OrderBy == OrderNew && rj.Query.FromDate.IsZero() {
		rj.Query.FromDate = now
	}

	return rj, nil
}

func (c *Client) Home(
	ctx context.Context,
	userID string,
) (*RecentJobs, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	return c.RecentJobs(ctx, &RecentJobsQuery{
		Amount:    50,
		JobType:   JobTypeNull,
		OrderBy:   OrderNew,
		JobStatus: JobStatusCompleted,
		UserID:    userID,
		Dedupe:    true,
	})
}

func (c *Client) CommunityFeed(ctx context.Context) (*RecentJobs, error) {
	return c.RecentJobs(ctx, &RecentJobsQuery{
		Amount:    50,
		JobType:   JobTypeUpscale,
		OrderBy:   OrderHot,
		JobStatus: JobStatusCompleted,
		Dedupe:    true,
	})
}

func (c *Client) PersonalFeed(ctx context.Context) (*RecentJobs, error) {
	return c.RecentJobs(ctx, &RecentJobsQuery{
		Amount:    50,
		JobType:   JobTypeUpscale,
		OrderBy:   OrderNew,
		JobStatus: JobStatusCompleted,
		Personal:  true,
		Dedupe:    true,
	})
}

func (c *Client) Bookmarks(
	ctx context.Context,
	userID string,
) (*RecentJobs, error) {
	if userID == "" {
		return nil, ErrUserIDRequired
	}

	return c.RecentJobs(ctx, &RecentJobsQuery{
		Amount:      50,
		JobType:     JobTypeNull,
		OrderBy:     OrderLikedTime,
		JobStatus:   JobStatusCompleted,
		UserIDLiked: userID,
		Dedupe:      true,
	})
}
