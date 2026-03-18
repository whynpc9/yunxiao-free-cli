# Search condition examples

Use these examples as starting points for `--conditions`.

## Project name contains text

```json
{
  "conditionGroups": [
    [
      {
        "fieldIdentifier": "name",
        "operator": "BETWEEN",
        "value": ["登录"],
        "toValue": null
      }
    ]
  ]
}
```

Example:

```bash
yx project search --conditions '{"conditionGroups":[[{"fieldIdentifier":"name","operator":"BETWEEN","value":["登录"],"toValue":null}]]}' --json
```

## Work item subject contains text

```json
{
  "conditionGroups": [
    [
      {
        "fieldIdentifier": "subject",
        "operator": "BETWEEN",
        "value": ["登录"],
        "toValue": null
      }
    ]
  ]
}
```

Example:

```bash
yx workitem search --category Req --space-id <projectId> --conditions '{"conditionGroups":[[{"fieldIdentifier":"subject","operator":"BETWEEN","value":["登录"],"toValue":null}]]}' --json
```

## Work item status in a set

```json
{
  "conditionGroups": [
    [
      {
        "fieldIdentifier": "status",
        "operator": "CONTAINS",
        "value": ["100005", "100010"],
        "toValue": null,
        "className": "status",
        "format": "list"
      }
    ]
  ]
}
```

## Test case title contains text

Some tenants use `subject`, others may expose a different visible field name in the UI. Start with `subject`; if it returns no results but data exists, inspect the raw JSON first.

```json
{
  "conditionGroups": [
    [
      {
        "fieldIdentifier": "subject",
        "operator": "BETWEEN",
        "value": ["登录"],
        "toValue": null
      }
    ]
  ]
}
```

Example:

```bash
yx testcase search --repo <testRepoId> --conditions '{"conditionGroups":[[{"fieldIdentifier":"subject","operator":"BETWEEN","value":["登录"],"toValue":null}]]}' --json
```
