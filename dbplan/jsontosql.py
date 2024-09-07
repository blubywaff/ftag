import json
from uuid import uuid4

def main():
    data = None
    with open("fullexport.json", "r", encoding="utf-8") as _f:
        data = json.loads(_f.read())
    if data is None:
        print("bad data")
        return
    seq = [0]
    def getseq(a):
        a[0] += 1
        return f"{a[0]:04d}"
    gsq = lambda : getseq(seq)
    query = ""
    for i, r in enumerate(data["resources"]):
        with open(f"files/{r['id']}", "rb") as _m:
            query += f"INSERT INTO Resource (id, mime, upload, data) VALUES ('{r['id']}', '{r['type']}', '{r['createdAt']}', '\\x{_m.read().hex()}'::bytea);\n"
            if i % 10 == 0:
                with open(f"fe/{gsq()}fullexport{i/10}.sql", "w", encoding="utf-8") as _f:
                    _f.write(query)
                query = ""
    with open(f"fe/{gsq()}fullexport_.sql", "w", encoding="utf-8") as _f:
        _f.write(query)
    query = ""
    for i, t in enumerate(data["tags"]):
        query += f"INSERT INTO Tag (id, name) VALUES ('{uuid4()}', '{t['name']}');\n"
    with open(f"fe/{gsq()}fullexportt.sql", "w", encoding="utf-8") as _f:
        _f.write(query)
    query = ""
    for r, ts in data["relations"].items():
        for t in ts:
            query += f"WITH tt (rid, tid) AS (SELECT '{r}'::uuid, id FROM Tag WHERE name = '{t}') INSERT INTO TagOn (resource_id, tag_id) SELECT rid, tid FROM tt;\n"
    with open(f"fe/{gsq()}fullexportr.sql", "w", encoding="utf-8") as _f:
        _f.write(query)


if __name__ == "__main__":
    main()
