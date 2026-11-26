-- 첫 스키마. 이후 변경은 이 파일을 고치지 말고 make migrate-create name=<변경내용> 으로 새 파일을 만든다.
-- 세션 설정(statement_timeout, timezone 등)은 앱이 연결할 때 건다 (internal/database/postgresql/database.go).

-- +goose Up
create schema if not exists v1;

create schema if not exists admin;

-- updated_at 자동 갱신. 테이블마다 trigger 를 걸어 쓴다.
-- +goose StatementBegin
create or replace function v1.set_updated_at() returns trigger as $$
begin
    new.updated_at = now();
    return new;
end;
$$ language plpgsql;
-- +goose StatementEnd

-- @title : API 에러 로그
create table v1.error_log (
    id bigint generated always as identity primary key,
    occurred_at timestamptz not null default now(),
    ip_address text not null,
    user_agent text not null,
    path text not null,
    http_method text not null,
    requested_url text not null,
    error_code integer not null,
    error_message text not null
);

create index error_log_occurred_at_idx on v1.error_log (occurred_at);

comment on table v1.error_log is 'API 에러 로그 (4xx 일부, 5xx)';

-- @title : 유저 테이블
-- @relation :
--      v1.users - v1.user_login_logs // 1:N 관계
create table v1.users (
    id bigint generated always as identity primary key,
    created_at timestamptz not null default now(),
    updated_at timestamptz not null default now(),
    email text not null,
    password_hash text not null
);

-- 이메일은 대소문자 구분 없이 유일하다. 앱은 소문자로 저장한다.
create unique index users_email_lower_key on v1.users (lower(email));

create trigger users_set_updated_at before update on v1.users
    for each row execute function v1.set_updated_at();

comment on table v1.users is '유저 테이블';
comment on column v1.users.id is '유저 고유 id';
comment on column v1.users.created_at is '생성 날짜';
comment on column v1.users.updated_at is '업데이트 날짜';
comment on column v1.users.email is '유저 이메일 (소문자)';
comment on column v1.users.password_hash is '유저 비밀번호 bcrypt 해시';

-- @title : 유저 로그인 로그 테이블
create table v1.user_login_logs (
    id bigint generated always as identity primary key,
    created_at timestamptz not null default now(),
    user_id bigint not null references v1.users (id) on delete cascade
);

create index user_login_logs_user_id_idx on v1.user_login_logs (user_id);

comment on table v1.user_login_logs is '유저 로그인 로그 테이블';
comment on column v1.user_login_logs.id is '로그인 로그 고유 id';
comment on column v1.user_login_logs.created_at is '로그인 날짜';
comment on column v1.user_login_logs.user_id is '로그인 유저 id';

-- +goose Down
drop table if exists v1.user_login_logs;
drop table if exists v1.users;
drop table if exists v1.error_log;
drop function if exists v1.set_updated_at();
drop schema if exists admin;
drop schema if exists v1;
