create schema godzilla collate utf8mb4_general_ci;

create table godzilla.scenario
(
    id         int auto_increment
        primary key,
    name       varchar(255)                        not null,
    definition longtext                            not null,
    created_at timestamp default CURRENT_TIMESTAMP null,
    updated_at timestamp default CURRENT_TIMESTAMP not null,
    constraint scenario_pk
        unique (name)
);

create table godzilla.job_status
(
    id          int auto_increment
        primary key,
    scenario_id int                                 not null,
    status      longtext                            not null,
    created_at  timestamp default CURRENT_TIMESTAMP null,
    updated_at  timestamp default CURRENT_TIMESTAMP not null,
    reason      text null
);
