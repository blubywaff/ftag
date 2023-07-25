from neo4j import GraphDatabase, RoutingControl
import shutil
import argparse

URI = "neo4j://127.0.0.1:7687"


def __get_ids(includes: list[str], excludes: list[str], exclude_mode: str) -> list[tuple[str, str]]:
    expart = ""
    if len(excludes) != 0:
        if exclude_mode == "or":
            expart = "AND none(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a)))"
        elif exclude_mode == "and":
            expart = "AND (NOT all(tag in $extag WHERE exists((:Tag {name: tag})-[:describes]->(a))))"
    with GraphDatabase.driver(URI, auth=None) as driver:
        records, _, _ = driver.execute_query(
            f"""
MATCH (a:Resource)
WHERE all(tag in $intag WHERE exists((:Tag {{name: tag}})-[:describes]->(a)))
{expart}
WITH a ORDER BY a.createdAt DESC, a.id ASC
RETURN a.id, a.type
""",
            routing_=RoutingControl.READ,
            extag=excludes,
            intag=includes,
        )
    return [(r["a.id"], r["a.type"]) for r in records]


def __copy_files(in_path: str, out_path: str, ids: list[tuple[str, str]]):
    if not in_path.endswith("/"):
        print("in_path should end with slash to represent directory")
        in_path += "/"
    if not out_path.endswith("/"):
        print("out_path should end with slash to represent directory")
        out_path += "/"
    for i, (id, rtyp) in enumerate(ids):
        shutil.copyfile(f"{in_path}{id}", f"{out_path}{i:0>-4d}-{id}.{rtyp.split('/')[1]}")


def __main():
    parser = argparse.ArgumentParser(
        prog="blubywaff ftag query export",
        description="exports all of the files that match a query, up to 1000",
        epilog="written by blubywaff for blubywaff's ftag system",
    )
    parser.add_argument("-I", "--inpath", action="store", required=True)
    parser.add_argument("-O", "--outpath", action="store", required=True)
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
    args = parser.parse_args()
    args_vars = vars(args)
    ids = __get_ids(args_vars["include"], args_vars["exclude"], args_vars["exmode"])
    __copy_files(args_vars["inpath"], args_vars["outpath"], ids)


if __name__ == "__main__":
    __main()
