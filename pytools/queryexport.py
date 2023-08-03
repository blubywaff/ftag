from neo4j import GraphDatabase, RoutingControl
import shutil
import argparse
import typing as typ
import json

T_id = str
T_mimetype = str
T_tagname = str


def __get_ids(
    vars: dict[str, typ.Any],
) -> list[tuple[T_id, T_mimetype, list[T_tagname]]]:
    excludes: str = vars["exclude"]
    includes: str = vars["include"]
    exclude_mode: str = vars["exmode"]
    max: int = vars["max"]
    creds: dict[str, str] = {}
    with open(vars["config"], "r", encoding="utf-8") as cf:
        creds = json.loads(cf.read())
    expart = ""
    if len(excludes) != 0:
        if exclude_mode == "or":
            expart = "AND none(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a)))"
        elif exclude_mode == "and":
            expart = "AND (NOT all(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a))))"
    with GraphDatabase.driver(creds["Neo4j"]["Url"], auth=(creds["Neo4j"]["Username"], creds["Neo4j"]["Password"])) as driver:  # type: ignore
        records, _, _ = driver.execute_query(
            f"""
MATCH (a:Resource)
WHERE all(tag in $intag WHERE exists((:Tag {{name: tag}})-[:describes]->(a)))
{expart}
WITH a ORDER BY a.createdAt DESC, a.id ASC LIMIT {max}
CALL {{
    WITH a
    MATCH (t:Tag)-[:describes]->(a)
    RETURN collect(t.name) as tn
}}
RETURN a, tn
""",  # type: ignore   This is a literalstring problem, but this script is only for priveleged users, and is caused by max, which is int, so no problem
            routing_=RoutingControl.READ,
            extag=excludes,
            intag=includes,
        )
    return [
        (r["a"]._properties["id"], r["a"]._properties["type"], r["tn"]) for r in records
    ]


def __copy_files(vars: dict[str, typ.Any], ids: list[tuple[str, str, list[str]]]):
    in_path: str = vars["inpath"]
    out_path: str = vars["outpath"]
    number: bool = vars["nums"]
    max: int = vars["max"]
    if not in_path.endswith("/"):
        print("in_path should end with slash to represent directory")
        in_path += "/"
    if not out_path.endswith("/"):
        print("out_path should end with slash to represent directory")
        out_path += "/"

    def fname(i: int, id: str, rtyp: str) -> str:
        return ("{i:0>-{width}d}-".format(width=len(str(max)), i=i) if number else "") + f"{id}.{rtyp.split('/')[1]}"

    # Write json manifest
    with open(out_path + "manifest.json", "w", encoding="utf-8") as mf:
        mf.write(
            json.dumps(
                {
                    fname(i, id, rtyp): {"id": id, "number": i, "type": rtyp, "tags": tags}
                    for i, (id, rtyp, tags) in enumerate(ids)
                }
            )
        )
    # Copy files
    for i, (id, rtyp, _) in enumerate(ids):
        shutil.copyfile(
            f"{in_path}{id}",
            f"{out_path}" + fname(i, id, rtyp),
        )


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
    parser.add_argument("-I", "--inpath", action="store", required=True, type=str)
    parser.add_argument("-O", "--outpath", action="store", required=True, type=str)
    parser.add_argument("-i", "--include", action="extend", nargs="+", required=True)
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
    ids = __get_ids(args_vars)
    __copy_files(args_vars, ids)


if __name__ == "__main__":
    __main()
