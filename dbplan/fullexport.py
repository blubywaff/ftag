from neo4j import GraphDatabase, RoutingControl
import shutil
import argparse
import typing as typ
import json

T_id = str
T_mimetype = str
T_tagname = str

def get_all(vars: dict[str, typ.Any]):
    creds: dict[str, str] = {}
    with open(vars["config"], "r", encoding="utf-8") as cf:
        creds = json.loads(cf.read())
    rsrc = []
    tag = []
    relation = {}
    with GraphDatabase.driver(creds["Neo4j"]["Url"], auth=(creds["Neo4j"]["Username"], creds["Neo4j"]["Password"])) as driver:  # type: ignore
        records, _a, _b = driver.execute_query("MATCH (n) RETURN n", routing_=RoutingControl.READ)
        for _r in records:
            r = _r["n"]
            if "Resource" in r.labels:
                rsrc.append({"id": r["id"], "createdAt": str(r["createdAt"]), "type": r["type"]})
            if "Tag" in r.labels:
                tag.append({"name": r["name"]})
        relat, _a, _b = driver.execute_query("MATCH (a:Tag)-->(b:Resource) RETURN a,b")
        for _r in relat:
            a, b = _r["a"], _r["b"]
            if b["id"] not in relation:
                relation[b["id"]] = []
            relation[b["id"]].append(a["name"])
    print("done", rsrc, tag, relation)
    return [rsrc, tag, relation]



def __args() -> argparse.ArgumentParser:
    """
    Include generated help as lazy substitute for documentation
    ```
    usage: queryexport.py [-h] -I INPATH -O OUTPATH -i INCLUDE [INCLUDE ...]
                          [-e EXCLUDE [EXCLUDE ...]] [-m {or,and}] [--max MAX]
                          [--nums | --no-nums] -c CONFIG

    exports all of the files that match a query, up to 1000 or number

    options:
      -h, --help            show this help message and exit
      -I INPATH, --inpath INPATH
      -O OUTPATH, --outpath OUTPATH
      -i INCLUDE [INCLUDE ...], --include INCLUDE [INCLUDE ...]
      -e EXCLUDE [EXCLUDE ...], --exclude EXCLUDE [EXCLUDE ...]
      -m {or,and}, --exmode {or,and}
                            Defaults to or. Should be explicity set if there is
                            more than one EXCLUDE
      --max MAX             Default 1000. Specifies the max number of results
      --nums, --no-nums     Specifies if the output files should be numbered
                            (default: True)
      -c CONFIG, --config CONFIG
                            The location of the ftag config file. Used for
                            database credentials.

    written by blubywaff for blubywaff's ftag system
    ```
    """
    parser = argparse.ArgumentParser(
        prog="queryexport.py",
        description="exports all of the files that match a query, up to 1000 or number",
        epilog="written by blubywaff for blubywaff's ftag system",
    )
    parser.add_argument("-I", "--inpath", action="store", required=False, type=str)
    parser.add_argument("-O", "--outpath", action="store", required=False, type=str)
    parser.add_argument("-i", "--include", action="extend", nargs="+", required=False)
    parser.add_argument(
        "-e", "--exclude", action="extend", nargs="+", default=[], required=False
    )
    parser.add_argument(
        "-m",
        "--exmode",
        action="store",
        choices=("or", "and"),
        default="or",
        required=False,
        help="Defaults to or. Should be explicity set if there is more than one EXCLUDE",
    )
    parser.add_argument(
        "--max",
        action="store",
        default=1000,
        required=False,
        help="Default 1000. Specifies the max number of results",
        type=int,
    )
    parser.add_argument(
        "--nums",
        action=argparse.BooleanOptionalAction,
        required=False,
        default=True,
        help="Specifies if the output files should be numbered",
    )
    parser.add_argument(
        "-c",
        "--config",
        action="store",
        required=True,
        help="The location of the ftag config file. Used for database credentials.",
        type=str,
    )
    return parser


def __main():
    args = __args().parse_args()
    args_vars = vars(args)
    res = get_all(args_vars)
    with open("fullexport.json", "w", encoding="utf-8") as _f:
        _f.write(json.dumps({"resources": res[0], "tags": res[1], "relations": res[2]}))

if __name__ == "__main__":
    __main()
