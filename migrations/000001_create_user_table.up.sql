create table users (
    login varchar(255) not null,
    password varchar(255) not null,
    created_at timestamp not null,
    primary key(login)
)