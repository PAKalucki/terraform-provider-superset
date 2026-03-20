create schema if not exists analytics;

create table if not exists analytics.events (
  id serial primary key,
  event_name text not null,
  created_at timestamp without time zone not null default now()
);

insert into analytics.events (event_name)
select 'provider_bootstrap'
where not exists (
  select 1
  from analytics.events
  where event_name = 'provider_bootstrap'
);
