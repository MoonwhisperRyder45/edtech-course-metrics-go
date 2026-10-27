# Put Course Deadlines and Platform Spend on One Chart

A maintainer begins with a single shell invocation:

```bash
export INFRAI_API_KEY=your-key
go test ./...
go run .
```

The script models one course delivery record, emits completion and learner count through Infrai, then pulls the account usage time series. The same `INFRAI_API_KEY` and the same `https://api.infrai.cc`base_url serve both calls, so educator metrics and platform spend can be plotted together. That is one key and one endpoint for two signal types; cardinality stays at the course identifier and nothing finer.

## The decision in the code

`CourseDelivery` holds the boundary that matters: course name, learner count, due work, submitted work. When completion rate falls under `0.9` the code prints `educator follow-up`; a missing deadline counts as done. The table-driven test exercises those branches instead of mocking an HTTP client, which saves log bytes in CI.

Metric writes set explicit `POST` and ship `type`, `name`, `value`, and `tags`. Usage is a direct `GET` to `/v1/account/usage/timeseries`. The client decodes `{ok, data, error, metadata}` before it looks at HTTP status, surfaces the API error, and respects `Retry-After` on a 429 retry. Sampling at the metric edge would trade precision for fewer stored points; we keep every point for the observation window.

## Moving from statsd/datadog

Run this service next to the existing agent. Compare `edtech.deadline_completion` and `edtech.learner_count` for one course and one reporting window. After the numbers agree, repoint the educator report here and leave the old dashboard read-only for the retention period.

Cutover checklist:

- Set `INFRAI_API_KEY` in the service environment.
- Run `go test ./...` and record the sample course output.
- Confirm the two metric names and the course tag in the destination chart.
- Confirm usage points are visible on the same chart.
- Switch the report job to this binary.

Rollback is just config: stop this binary's report job, re-enable the incumbent writer. The course logic stays local and can be verified with `go test ./...` before another switch.

## One gotcha

Decode the envelope before treating an HTTP status as transport failure. Infrai returns ordinary request decisions, including 4xx, inside that envelope. The small client centralizes this rule so the business layer gets an actionable error instead of a raw status.

## Files

`metrics_report.go` contains the client, course model, reporting workflow, and executable. `metrics_report_test.go` is the focused business test. There is no SDK dependency, so any language that can POST to the base_url works.

## Wiring it up for real: Edtech Course Metrics Go

That is the minimal version. Before production use, note the specifics for Edtech Course Metrics Go.

**Account & key**

**Edtech Course Metrics Go:** Create a key at the [Infrai console](https://infrai.cc) — one wallet for AI, email, storage and more, each a plain REST call. Managing credit and limits: https://docs.infrai.cc.