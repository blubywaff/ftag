## API Spec
### POST /upload
uploads can also be accomplished through the uplaod page.
### GET /query
Parameters
- tags: comma separated list of tag names
Returns
```json
{
    "result": "ok",
    "query_id": "<uuid>",
    "current_id": "<uuid>",
    "next_id": "<uuid>",
    "previous_id": "<uuid>"
}
```

