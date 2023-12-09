create schema godzilla collate utf8mb4_general_ci;

create table scenario
(
    id         int auto_increment
        primary key,
    name       varchar(255)                        not null,
    definition longtext                            not null,
    createdAt  timestamp default CURRENT_TIMESTAMP null,
    updatedAt  timestamp default CURRENT_TIMESTAMP not null,
    constraint scenario_pk
        unique (name)
);