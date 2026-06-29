create table if not exists mcp_http_events (
  session_id text not null references mcp_http_sessions(id) on delete cascade,
  sequence bigint not null,
  event_id text not null,
  payload bytea not null,
  created_at timestamptz not null default now(),
  primary key (session_id, sequence),
  unique (session_id, event_id)
);

create index if not exists mcp_http_events_created_idx
  on mcp_http_events(created_at);
