# Session evidence

The command gh api repos/example/service/actions/runs returned HTTP 503 from
the external service. A bounded retry succeeded without changing repository
files. The repository check then passed.

No repository-owned cause or unresolved workaround is present.
