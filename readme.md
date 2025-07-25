`ftag` web code, written in golang.

Copyright blubywaff 2025

# What?
ftag is a system that allows storage of files with a number of tags or attributes that can be used to search and sort these files.

# Current Stage
In its current stage, it is a single user system, designed to be locked behind some external firewall or authentication method.
Additionally, its current stage is mostly for demonstration, so it primarily supports web-viewable content (image, video).

# Tech Stack
## Graph Database (JanusGraph)
A graph database is used to store a reference of the files (resources) stored and the tags associated with each one.
A graph database is really the bread and butter of the logic for this system.
I used to use Neo4j but it starting making things difficult so now it supports any tinkerpop system (hopefully).
Currently configured to use Janusgraph

## Golang
Written in Golang for backend and attempting to keep the dependencies relatively minimal.

## Frontend
The frontend is built with svelte and is designed to be served statically and content produced using the API.
It also uses TailwindCSS because I like it.

# Running
```shell
./build.sh
docker compose up -d
./ftag
```
