---
title: Loops
subtitle: For loops and iteration
layout: page
---

Use `for:` in steps to invoke tasks repeatedly with different loop variables.

## Examples

@tabs
@file "List Loop" loops/list.yml
@file "Nested Loop" loops/nested.yml
@file "Matrix Loop" loops/matrix.yml

![](./loops/list.png)

## How It Works

1. **`for:` in a step** - defines the loop with `for: variable in collection`
2. **`task:` in the same step** - names the task to invoke for each iteration
3. **Loop variable** - becomes available in the invoked task as `${{ variable }}`
4. **`requires:`** - the invoked task can declare required variables to validate they are present

## Missing Variables

If a required variable is missing, execution fails with a clear error:

```text
job 'deploy_service' requires variables [env service_version] but missing: [env]
```

## See Also

- [Steps](./steps) - Step configuration
- [Jobs](./jobs) - Job configuration
