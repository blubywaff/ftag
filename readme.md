`ftag` web code, written in golang.

Copyright blubywaff 2023

# What?
ftag is a system that allows storage of files with a number of tags or attributes that can be used to search and sort these files.

# Current Stage
In its current stage, it is a single user system, designed to be locked behind some external firewall or authentication method.
Additionally, its current stage is mostly for demonstration, so it primarily supports web-viewable content (image, video).

# Tech Stack
## Neo4j
Neo4j is used to store a reference of the files (resources) stored and the tags associated with each one.
A graph database is really the bread and butter of the logic for this system, and Neo4j with Cypher has made this a lot easier for me.

## Golang
Written in Golang for backend and using go templates (`html/template`) for server rendering of frontend.

## Frontend
The front end is designed to be fully operable without javascript.
This means every page is built using html and css, and that interaction is driven primarily by html forms.

## TailwindCSS
Tailwind has massively sped up development of webpages while keeping the code easy to manage.
This was my first time using Tailwind and I have some repeated components that could be dealt with better.

