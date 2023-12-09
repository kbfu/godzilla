create schema godzilla collate utf8mb4_general_ci;

create table godzilla.repo
(
    id      int auto_increment
        primary key,
    address varchar(255) null,
    name    varchar(255) not null,
    branch  varchar(255) not null
);