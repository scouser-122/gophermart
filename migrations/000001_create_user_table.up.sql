CREATE TABLE users (
    login varchar(255) NOT NULL,
    password varchar(255) NOT NULL,
    balance numeric(30,2) DEFAULT 0.0,
    created_at timestamp NOT NULL,
    PRIMARY KEY(login)
)