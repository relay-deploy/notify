create table event_cursors (
    source text primary key,
    transaction_id bigint not null,
    sequence bigint not null
);

create table crash_alerts (
    app_id uuid primary key,
    project_id uuid not null,
    release_id uuid not null,
    opened_at timestamptz not null,
    last_crash_at timestamptz not null,
    crash_count int not null,
    last_exit_code int not null,
    last_out_of_memory boolean not null,
    ran_out_of_memory boolean not null,
    last_notified_at timestamptz not null,
    last_notified_crash_count int not null,
    recovered_at timestamptz
);

create table live_releases (
    app_id uuid primary key,
    release_id uuid not null
);

create table emails (
    id uuid primary key default uuidv7(),
    kind text not null,
    app_id uuid not null,
    project_id uuid not null,
    facts jsonb not null,
    subject text,
    html text,
    text_body text,
    status text not null default 'pending',
    attempts int not null default 0,
    occurred_at timestamptz not null,
    next_attempt_at timestamptz not null,
    created_at timestamptz not null,
    sent_at timestamptz,
    last_error text,
    provider_id text
);

create index emails_pending_next_attempt_at on emails (next_attempt_at) where status = 'pending';
