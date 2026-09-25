# ADR-0001: Keep exactly 5 columns for the Flow Table

- Status: accepted
- Date: 2026-09-21

## Context

The Flow Table has 5 columns: id, type, parent, content, metadata. The metadata
column carries the required references from, to and attach, alongside optional keys
such as style and back.

Counting over the specification's 22-row example: id and type are present in 100% of
rows, content in 77%, parent in 50%, from and to in 36%, attach in 9%.

That reveals an asymmetry. The parent column is empty exactly on edge rows, while from
and to, which are required references, are hidden in a string meant for optional data.
xlsx users cannot filter or sort by from and to, and checking for dangling references
has to go through the metadata parsing layer.

The deletion test shows that the column versus metadata boundary is not a modeling
problem: dropping any column only moves its content into metadata, and the model and
validation rules stay the same. This is purely about the ergonomics of whoever writes
the table.

Option considered and rejected: go to 7 columns (id, type, parent, from, to, content,
metadata), with a compatibility reader that lifts from and to out of metadata into
columns so old files keep working.

## Decision

Keep exactly 5 columns. Do not add from and to columns.

Explicitness moves into a metadata schema declared by the kind: per type, which keys
are required, which keys are references to other ids, and the value range of each key.

## Consequences

- The number of columns is a compatibility commitment. Every table already written,
  every existing .csv and .xlsx file, and the prompt of the flowtable-drawio subagent
  need no changes.
- Checking for dangling references becomes a loop over the keys marked is_ref in the
  schema, instead of handwriting each case in validate.
- The web service can return errors at the level of individual keys in a metadata cell,
  thanks to the schema and to the structured Location.
- Adding a new diagram type never requires adding columns: a kind extends the schema,
  not the table.
- In exchange, xlsx users still cannot filter by from and to. This is a known and
  accepted cost.
