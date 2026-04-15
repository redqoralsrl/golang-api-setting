-- postgresql setting
SET
    idle_in_transaction_session_timeout = '30s';

SET
    idle_session_timeout = '10min';

SET
    statement_timeout = '10s';

SET
    lock_timeout = '5s';

SET
    TIME ZONE 'UTC';

-- postgresql schema 분리
create schema v1;

create schema admin;

-- @title : API 에러 로그
create table
    v1.error_log (
        id serial primary key,
        timestamp timestamp not null,
        ip_address varchar(255) not null,
        user_agent varchar(255) not null,
        path varchar(255) not null,
        http_method varchar(255) not null,
        requested_url text not null,
        error_code integer not null,
        error_message text not null
    );

-- @title : 유저 테이블
-- @relation :
--      v1.users - v1.user_logs // 1:N 관계
create table
    v1.users (
        id serial primary key, -- 유저 고유 ID
        created_at timestamptz not null default current_timestamp, -- 유저 생성 시간
        updated_at timestamptz not null default current_timestamp, -- 유저 업데이트 시간
        email varchar(255) unique not null, -- 유저 이메일
        password_hash text not null -- 유저 비밀번호
    );

comment on table v1.users is '유저 테이블';

comment on column v1.users.id is '유저 고유 id';

comment on column v1.users.created_at is '생성 날짜';

comment on column v1.users.updated_at is '업데이트 날짜';

comment on column v1.users.email is '유저 이메일';

comment on column v1.users.password_hash is '유저 비밀번호';

-- @title : 유저 로그인 로그 테이블
-- @relation :
--      v1.users - v1.user_logs // 1:N 관계
create table
    v1.user_login_logs (
        id serial primary key, -- 로그인 로그 고유 ID
        created_at timestamptz not null default current_timestamp, -- 로그인 시도 시간
        user_id integer references v1.users (id) not null -- 로그인 시도 유저 ID
    );

comment on table v1.user_login_logs is '유저 로그인 로그 테이블';

comment on column v1.user_login_logs.id is '로그인 로그 고유 id';

comment on column v1.user_login_logs.created_at is '로그인 시도 날짜';

comment on column v1.user_login_logs.user_id is '로그인 시도 유저 id';
