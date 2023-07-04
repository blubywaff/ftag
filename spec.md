## Database Spec
### Graph Structure
`(t:Tag)`
- `name <string>` must contain only characters `[a-z\-]` and is not case sensitive
- `createdAt <datetime>`
`(r:Resource)`
- `identity <uuid>`
- `uploadedAt <datetime>`
- `type <mimetype>`
`(:Tag)-[d:describes]->(:Resource)`
- `describes` connect tags to resources
`(:Resource {type: "application/vnd.blubywaff.collection"})-[c:contains {number: <int>}]->(:Resource)`
- `contains` specify the member resources for a collection
- member resources (right side of contains) should on be connected to tags if they are meant to be directly referenced

### Types
For now, the table only includes non standard or unusual mime types.
| MIME Type | Description |
|-----------|-------------|
| `application/vnd.blubywaff.collection` | a resource that represents an ordered collection of other resources |


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

