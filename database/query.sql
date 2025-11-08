-- name: ListFeeds :many
select
  *
from
  feed;

-- name: CreateFeed :exec
insert into
  feed (name, url, feed_type)
values
  (?, ?, ?);

-- name: UpdateFeedDate :exec
update feed
set
  last_updated_at = ?
where
  id = ?;

-- name: DeleteFeed :exec
delete from feed
where
  id = ?;

-- name: ListPost :many
select
  *
from
  post;

-- name: DeletePost :exec
delete from post
where
  id = ?;
