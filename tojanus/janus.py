import shutil
from gremlin_python.process.anonymous_traversal import traversal
from gremlin_python.process.graph_traversal import  __
from gremlin_python.driver.driver_remote_connection import DriverRemoteConnection
from gremlin_python.process.traversal import eq
import json

def main():
    gremlin_url = ''  # 'ws://localhost:8182/gremlin'
    prefix = '.'
    files = ''
    g = traversal().with_remote(DriverRemoteConnection(gremlin_url,'g'))
    with open(f"{prefix}/export.json", 'r', encoding='utf-8') as f:
        resources = json.load(f)
    notag = []
    tags = set()
    tagmap = []
    for r in resources:
        notag.append({k: v for k, v in r.items() if k != 'tags'})
    for r in resources:
        for t in r["tags"]:
            tags.add(t)
            tagmap.append({'r': r["id"], 't': t})
    tags = list(tags)

    batch = 500
    for idx in range(0, len(tags), batch):
        g.inject(tags[idx:idx+batch]).unfold().as_('tn') \
        .addV('tag').property('name', __.select('tn')) \
        .iterate()
        print(f"progress: tags   (1/3) : {idx} / {len(tags)}")
    print(f"progress: tags   (1/3)")
    for idx in range(0, len(notag), batch):
        g.inject(notag[idx:idx+batch]).unfold().as_('rss_dct') \
        .addV('resource') \
        .property("rsc_id", __.select('rss_dct').select('id')) \
        .property("mime", __.select('rss_dct').select('mime')) \
        .property("upload", __.select('rss_dct').select('upload')) \
        .iterate()
        print(f"progress: notag  (2/3) : {idx} / {len(notag)}")
    print(f"progress: notag  (2/3)")
    for idx in range(0, len(tagmap), batch):
        g.inject(tagmap[idx:idx+batch]).unfold().as_('tm') \
        .select('t').as_('t') \
        .select('tm').select('r').as_('r') \
        .V().as_('v').hasLabel('tag').values('name').where(eq('t')) \
        .select('v').as_('t') \
        .V().as_('v').hasLabel('resource').values('rsc_id').where(eq('r')) \
        .select('v').as_('r') \
        .addE('describes').from_('t').to('r') \
        .iterate()
        print(f"progress: tagmap (3/3) : {idx} / {len(tagmap)}")
    print(f"progress: tagmap (3/3)")

    print(f"progress: copy files (0/{len(resources)})")
    for i, r in enumerate(resources):
        if i % 5 == 0:
            print(".", end="", flush=True)
        if i % 100 == 0:
            print(f"progress: copy files ({i}/{len(resources)})")
        in_file = f"{prefix}/{r['id']}.{r['mime'].split('/')[-1]}"
        out_file = f"{files}/{r['id']}"
        shutil.copy(in_file, out_file)
    print(f"progress: copy files ({len(resources)})")


if __name__ == "__main__":
    main()
