CREATE TABLE users (
    login varchar(255) NOT NULL,
    password varchar(255) NOT NULL,
    balance double precision DEFAULT 0.0,
    created_at timestamp NOT NULL,
    PRIMARY KEY(login)
)