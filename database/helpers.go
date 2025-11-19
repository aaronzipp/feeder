package database

import (
	"context"
	"database/sql"
)

// PostWithFeed is an alias for the unified post with feed type
type PostWithFeed = ListPostsWithFeedFilteredRow

// ListInbox returns all non-archived posts with feed information
func (q *Queries) ListInbox(ctx context.Context) ([]PostWithFeed, error) {
	return q.ListPostsWithFeedFiltered(ctx, ListPostsWithFeedFilteredParams{
		IsArchived: sql.NullInt64{Int64: 0, Valid: true},
		IsStarred:  nil, // No filter on starred
	})
}

// ListArchive returns all archived posts with feed information
func (q *Queries) ListArchive(ctx context.Context) ([]PostWithFeed, error) {
	return q.ListPostsWithFeedFiltered(ctx, ListPostsWithFeedFilteredParams{
		IsArchived: sql.NullInt64{Int64: 1, Valid: true},
		IsStarred:  nil, // No filter on starred
	})
}

// ListStarred returns all starred posts with feed information
func (q *Queries) ListStarred(ctx context.Context) ([]PostWithFeed, error) {
	return q.ListPostsWithFeedFiltered(ctx, ListPostsWithFeedFilteredParams{
		IsArchived: nil, // No filter on archived
		IsStarred:  sql.NullInt64{Int64: 1, Valid: true},
	})
}
