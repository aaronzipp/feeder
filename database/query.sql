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

-- name: UpdateFeedFormat :exec
update feed
set
  date_format = ?
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

-- name: CreatePost :exec
insert
or ignore into post (title, url, published_at, feed_id)
values
  (?, ?, ?, ?);

-- name: DeletePost :exec
delete from post
where
  id = ?;

-- name: ListPostsWithFeed :many
select
  p.id,
  p.title,
  p.url,
  p.published_at,
  p.feed_id,
  f.name as feed_name
from
  post p
  inner join feed f on p.feed_id = f.id
order by
  p.published_at desc;

-- name: ListInbox :many
select
  p.id,
  p.title,
  p.url,
  p.published_at,
  p.feed_id,
  f.name as feed_name
from
  post p
  inner join feed f on p.feed_id = f.id
where
  not is_archived
  and not is_starred
order by
  p.published_at desc;

-- name: ListArchive :many
select
  p.id,
  p.title,
  p.url,
  p.published_at,
  p.feed_id,
  f.name as feed_name
from
  post p
  inner join feed f on p.feed_id = f.id
where
  is_archived
  and not is_starred
order by
  p.published_at desc;

-- name: ListStarred :many
select
  p.id,
  p.title,
  p.url,
  p.published_at,
  p.feed_id,
  f.name as feed_name
from
  post p
  inner join feed f on p.feed_id = f.id
where
  is_starred
order by
  p.published_at desc;

-- ArchivePost :exec
update post
set
  is_archived = 1
where
  id = ?;

-- UnarchivePost :exec
update post
set
  is_archived = 0
where
  id = ?;

-- StarPost :exec
update post
set
  is_starred = 1
where
  id = ?;

-- UnstarPost :exec
update post
set
  is_starred = 0
where
  id = ?;
