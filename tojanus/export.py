import psycopg
from psycopg.rows import dict_row
import json
import base64

conn_params = {
    "host": "localhost",
    "port": 5432,
    # "database": "postgres",
    "user": "postgres",
    "password": "postgres",
}


def get_resources(conn):
    with psycopg.ServerCursor(conn, "rsc_cursor") as cur:
        cur.execute("SELECT id,mime,upload,data FROM resource")

        for row in cur:
            with psycopg.ServerCursor(conn, "tag_cursor") as tag_cur:
                tag_cur.execute("SELECT name FROM rtags WHERE id = %s", [row["id"]])
                rows = tag_cur.fetchall()
                result = {
                    "id": str(row["id"]),
                    "mime": str(row["mime"]),
                    "upload": row["upload"].isoformat(),
                    "tags": list(map(lambda x: x["name"], rows)),
                }
                yield result, row["data"]


def main():
    prefix = "."
    with psycopg.connect(**conn_params, row_factory=dict_row) as conn:
        resources = get_resources(conn)
        agg = []
        for rsc, dat in resources:
            agg.append(rsc)
            with open(f'{prefix}/{rsc["id"]}.{rsc["mime"].split("/")[-1]}', "wb") as f:
                f.write(dat)
        with open(f"{prefix}/export.json", "w", encoding="utf-8") as f:
            f.write(json.dumps(agg))


if __name__ == "__main__":
    main()
