# Apollo
> Program for easily querying and collecting EVM chaindata based on a schema.

## To Do
- [ ] How to best generate dynamic golang code (based on schema)
  - [ ] Look at custom struct tags
  - [ ] Should probably not generate golang code but just ABI pack values into transaction input field
- [x] How to best generate SQL DDL based on schema
  - Should we use an ORM package or just plain SQL?
  - For now, just plain SQL will do
- [x] Simplify schema parser
  - [x] Upgrade schema to yaml

## Notes
* Maybe in the future we could bypass the schema and just create an SQL chain query directly.
Think of how Dune Analytics does it