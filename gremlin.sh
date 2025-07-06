#!/bin/bash
docker run --rm -it --network=host janusgraph/janusgraph:latest bin/gremlin.sh
# :remote connect tinkerpop.server conf/remote.yaml
# :remote console
