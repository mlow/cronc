# cronc
A cron scheduler for containers.

`cronc` is an alternative to classic cron schedulers (`cron`, `cronie`,
`anacron`, ...) suited for container environments.

The key motivation for its creation is that classical cron schedulers are not
designed to pass their own environments on to their jobs. On a system with many
scheduled jobs, this makes sense - each job should be responsible its own
environment environment. However, in a container, one might only be scheduling a
small handful of jobs relating to a single application. In such a case it would
be convenient of the container's environment were available to the jobs.

A common way to achieve this is to have a custom entrypoint which saves
environment variables to a file somewhere in the container. Jobs can then source
those variables when they run. Another option is to save them to
`/etc/environment` so they are automatically available to jobs.

Both of those options feel wrong to me, and so, here's `cronc`.

In summary, `cronc`:
  - Passes its own environment through to jobs.
  - Redirects the `stdout`/`stderr` of jobs its own `stdout`/`stderr`.
  - Reads cron tabs from the following sources:
    - The file or directory given via the `--cronPath` option (default: `/dev/crontab`)
      - If the path is a directory, all files in that directory will be read
    - The environment variables given via the `--cronVar` option (default: `CRONTAB`)
  - Does not support setting a per-job user - all jobs are run as the container user.
  - Does not support setting environment variables directly in the crontab
  - Runs in the foreground by default and responds appropriately to signals.
