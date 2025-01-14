## Setup

These notes are a stream of consiousness as I've gotten this working end-to-end. Will need to be revised to make it
repeatable.

Clickhouse creates a lot of files so put it in a place we're okay with that stuff existing:

```
; mkdir clickhouse
; cd clickhouse
; curl https://clickhouse.com/ | sh
; ./clickhouse server
```

Create the database for our telemetry to flow into. The remainder of the schema will be created lazily:

```
; ./clickhouse client
ClickHouse client version 25.1.1.1857 (official build).
Connecting to localhost:9000 as user default.
Connected to ClickHouse server version 25.1.1.

Warnings:
 * Maximum number of threads is lower than 30000. There could be problems with handling a lot of simultaneous queries.

fortitude.local :) create database otel;

CREATE DATABASE otel

Query id: 5fc19c4b-f913-4cfc-b56b-d5fd662304cd

Ok.

0 rows in set. Elapsed: 0.003 sec.
```

Build and run the OpenTelemetry Collector:

```
; make run
```

At this point any OTLP received will be batched and sent to ClickHouse. Do this with `sampledatagen`:

```
; go run ./cmd/sampledatagen
```

We can now query these back from ClickHouse:

```
SELECT *
FROM otel.otel_traces

Query id: f9549204-0c17-4588-bc02-8bd3f85e4658

Connecting to localhost:9000 as user default.
Connected to ClickHouse server version 25.1.1.

Row 1:
──────
Timestamp:          2025-01-07 15:29:54.000000000
TraceId:            6c1946a9d562bb820c0801ef1677ed26
SpanId:             4d26b9f6c7d72fa5
ParentSpanId:
TraceState:
SpanName:           go_modules in  for github.com/go-git/go-git/v5 - Update #943765154
SpanKind:           Server
ServiceName:        flipt-io-flipt-private
ResourceAttributes: {'cicd.github.workflow.run.attempt':'7','cicd.github.workflow.run.conclusion':'success','cicd.github.workflow.run.created_at':'2025-01-06T16:32:19Z','cicd.github.workflow.run.id':'12636346856','cicd.github.workflow.run.name':'go_modules in  for github.com/go-git/go-git/v5 - Update #943765154','cicd.github.workflow.run.started_at':'2025-01-07T20:29:54Z','cicd.github.workflow.run.status':'completed','cicd.github.workflow.run.updated_at':'2025-01-07T20:30:13Z','cicd.github.workflow.run.url':'https://github.com/flipt-io/flipt-private/actions/runs/12636346856','cicd.pipeline.name':'go_modules in  for github.com/go-git/go-git/v5 - Update #943765154','cicd.pipeline.run.id':'12636346856','scm.git.head.branch':'main','scm.git.head.commit.author.email':'49699333+dependabot[bot]@users.noreply.github.com','scm.git.head.commit.author.login':'','scm.git.head.commit.author.name':'dependabot[bot]','scm.git.head.commit.message':'chore(deps): bump github.com/aws/aws-sdk-go-v2/config (#336)\n\nBumps [github.com/aws/aws-sdk-go-v2/config](https://github.com/aws/aws-sdk-go-v2) from 1.27.27 to 1.28.7.\n- [Release notes](https://github.com/aws/aws-sdk-go-v2/releases)\n- [Commits](https://github.com/aws/aws-sdk-go-v2/compare/config/v1.27.27...config/v1.28.7)\n\n---\nupdated-dependencies:\n- dependency-name: github.com/aws/aws-sdk-go-v2/config\n  dependency-type: direct:production\n  update-type: version-update:semver-minor\n...\n\nSigned-off-by: dependabot[bot] <support@github.com>\nCo-authored-by: dependabot[bot] <49699333+dependabot[bot]@users.noreply.github.com>','scm.git.head.commit.sha':'154ed2aea45dcef1b5ef2cb78a4725e5159bb7ca','scm.git.repository.id':'820343353','scm.git.repository.name':'flipt-private','scm.git.repository.owner':'flipt-io','scm.git.repository.url':'https://github.com/flipt-io/flipt-private','service.name':'flipt-io-flipt-private','vcs.repository.ref.name':'main','vcs.repository.ref.revision':'154ed2aea45dcef1b5ef2cb78a4725e5159bb7ca','vcs.repository.url.full':'https://github.com/flipt-io/flipt-private'}
ScopeName:
ScopeVersion:
SpanAttributes:     {}
Duration:           19000000000 -- 19.00 billion
StatusCode:         Ok
StatusMessage:      success
Events.Timestamp:   []
Events.Name:        []
Events.Attributes:  []
Links.TraceId:      []
Links.SpanId:       []
Links.TraceState:   []
Links.Attributes:   []

Row 2:
──────
Timestamp:          2025-01-07 15:30:06.000000000
TraceId:            6c1946a9d562bb820c0801ef1677ed26
SpanId:             ff79418cbec27198
ParentSpanId:
TraceState:
SpanName:           Dependabot
SpanKind:           Server
ServiceName:        flipt-io-flipt-private
ResourceAttributes: {'cicd.github.workflow.job.completed_at':'2025-01-07T20:30:10Z','cicd.github.workflow.job.conclusion':'success','cicd.github.workflow.job.created_at':'2025-01-07T20:29:57Z','cicd.github.workflow.job.id':'35276613318','cicd.github.workflow.job.name':'Dependabot','cicd.github.workflow.job.started_at':'2025-01-07T20:30:06Z','cicd.github.workflow.job.status':'completed','cicd.github.workflow.job.url':'https://github.com/flipt-io/flipt-private/actions/runs/12636346856/job/35276613318','cicd.pipeline.task.name':'Dependabot','cicd.pipeline.task.run.id':'35276613318','cicd.pipeline.task.run.url.full':'https://github.com/flipt-io/flipt-private/actions/runs/12636346856/job/35276613318','scm.git.head.branch':'main','scm.git.head.commit.sha':'154ed2aea45dcef1b5ef2cb78a4725e5159bb7ca','scm.git.repository.id':'820343353','scm.git.repository.name':'flipt-private','scm.git.repository.owner':'flipt-io','scm.git.repository.url':'https://github.com/flipt-io/flipt-private','service.name':'flipt-io-flipt-private','vcs.repository.ref.name':'main','vcs.repository.ref.revision':'154ed2aea45dcef1b5ef2cb78a4725e5159bb7ca','vcs.repository.url.full':'https://github.com/flipt-io/flipt-private'}
ScopeName:
ScopeVersion:
SpanAttributes:     {}
Duration:           4000000000 -- 4.00 billion
StatusCode:         Unset
StatusMessage:
Events.Timestamp:   []
Events.Name:        []
Events.Attributes:  []
Links.TraceId:      []
Links.SpanId:       []
Links.TraceState:   []
Links.Attributes:   []

2 rows in set. Elapsed: 0.006 sec.
```
