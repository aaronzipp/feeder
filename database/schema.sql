create table feed (
  id integer primary key,
  name text not null,
  last_updated_at text,
  url text not null,
  feed_type text check (feed_type in ('rss', 'atom', 'custom')) not null,
  date_format text
);

create table post (
  id integer primary key,
  title text not null,
  url text not null,
  published_at text not null,
  feed_id integer not null,
  is_archived integer default 0,
  is_starred integer default 0,
  foreign key (feed_id) references feed (id) on delete cascade,
  unique (url, feed_id)
);
