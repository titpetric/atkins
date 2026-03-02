# Skills and Workspaces

Atkins loads skills from three locations in order:

1. First detected main pipeline (optional)
2. First detected `.atkins/skills/` folder (optional)
3. Global `~/.atkins/skills/` (optional)

Local skills take precedence over global skills with the same filename.

## Skill Matching with `when:`

Skills can specify files that must exist to enable the skill:

```yaml
when:
  files:
    - compose.yml
    - docker-compose.yml
```

Atkins searches from the current directory upward. If any listed file is found, the skill is enabled and its working directory is set to the folder containing that file.

This allows a compose skill to run `docker compose` commands from the correct path regardless of where you invoke atkins.

## Workspace Skills

A workspace skill in `[project]/.atkins/skills/deploy.yml` applies to the entire project.

**With `when:`** - working directory is set to the folder containing the matched file.

**Without `when:`** - working directory is set to the folder containing `.atkins/`. This allows the skill to run project-level commands from the workspace root.

## Global Skills

Global skills in `~/.atkins/skills/` are available everywhere without populating your source tree.

Global skills do not change the working directory. They run from wherever you invoke atkins, unless:

- They have a `when:` that matches a file (uses that file's folder)
- They explicitly set `dir:` in the pipeline

## Project Structure

The main pipeline and `.atkins/` folder define workspace boundaries. Example:

```
/project/.atkins/          # workspace skills
/project/atkins.yml        # main pipeline
/project/app/compose.yml   # matched by compose skill
/project/app/sub/          # can invoke skills from here
```

From `/project/app/sub/`, atkins searches upward to find configuration and skills. A compose skill with `when: files: [compose.yml]` would match `/project/app/compose.yml` and run from `/project/app/`.

Nested `.atkins/` folders or pipelines create separate workspaces with their own scope.
