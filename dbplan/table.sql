DROP TABLE IF EXISTS TagOn;
DROP TABLE IF EXISTS Tag;
DROP TABLE IF EXISTS Resource;

CREATE TABLE Resource (
    id uuid PRIMARY KEY,
    mime varchar(255) NOT NULL,
    upload timestamp NOT NULL,
    data bytea NOT NULL
);

CREATE TABLE Tag (
    id uuid PRIMARY KEY,
    name varchar(31) UNIQUE NOT NULL
);

CREATE TABLE TagOn (
    tag_id uuid NOT NULL REFERENCES Tag (id),
    resource_id uuid NOT NULL REFERENCES Resource (id),
    PRIMARY KEY (tag_id, resource_id)
);
