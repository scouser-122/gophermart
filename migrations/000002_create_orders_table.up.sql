CREATE TABLE orders (
    id varchar(255) NOT NULL,
    status varchar(20) NOT NULL,
    uploaded_at timestamp NOT NULL,
    accrual bigint NULL DEFAULT NULL,
    user_login varchar(255) NOT NULL,
    PRIMARY KEY(id),
    FOREIGN KEY (user_login) REFERENCES users (login)
);