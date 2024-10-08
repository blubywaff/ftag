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

CREATE OR REPLACE VIEW rtags AS
SELECT resource.*, tag.name
FROM resource, tag, tagon
WHERE resource.id = tagon.resource_id
    AND tag.id = tagon.tag_id;

CREATE OR REPLACE FUNCTION tagquery(includes TEXT[], excludes TEXT[], off INT, lim INT)
RETURNS TABLE(id uuid, upload timestamp, mime varchar(255))
AS $$
BEGIN
    RETURN QUERY
    SELECT DISTINCT tt.id, tt.upload, tt.mime
    FROM (
        SELECT rtags.id, rtags.upload, rtags.mime
        FROM rtags
        WHERE rtags.name = any (includes)
        OR ARRAY_LENGTH(includes, 1) IS NULL
        GROUP BY rtags.id, rtags.upload, rtags.mime
        HAVING COUNT(rtags.name) = COALESCE(ARRAY_LENGTH(includes, 1), COUNT(rtags.name))
        EXCEPT
        SELECT rtags.id, rtags.upload, rtags.mime
        FROM rtags
        WHERE name = any (excludes)
    ) as tt
    ORDER BY tt.upload DESC, tt.id ASC
    LIMIT lim
    OFFSET off;
END;
$$ LANGUAGE PLPGSQL;

