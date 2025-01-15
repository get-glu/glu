package githubreceiver

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/get-glu/glu/otel/internal/ids"

	"github.com/google/go-github/v68/github"
	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/ptrace"
	semconv "go.opentelemetry.io/otel/semconv/v1.27.0"
)

const (
	GitHubWorkflowRunConclusionKey = "cicd.github.workflow.run.conclusion"
	GitHubWorkflowRunCreatedAtKey  = "cicd.github.workflow.run.created_at"
	GitHubWorkflowRunStartedAtKey  = "cicd.github.workflow.run.started_at"
	GitHubWorkflowRunUpdatedAtKey  = "cicd.github.workflow.run.updated_at"
	GitHubWorkflowRunAttemptKey    = "cicd.github.workflow.run.attempt"
	GitHubWorkflowRunStatusKey     = "cicd.github.workflow.run.status"
	GitHubWorkflowRunIDKey         = "cicd.github.workflow.run.id"
	GitHubWorkflowRunNameKey       = "cicd.github.workflow.run.name"
	GitHubWorkflowRunURLKey        = "cicd.github.workflow.run.url"
	GitHubWorkflowRunActorIDKey    = "cicd.github.workflow.run.actor.id"
	GitHubWorkflowRunActorLoginKey = "cicd.github.workflow.run.actor.login"

	GitHubWorkflowJobConclusionKey  = "cicd.github.workflow.job.conclusion"
	GitHubWorkflowJobCreatedAtKey   = "cicd.github.workflow.job.created_at"
	GitHubWorkflowJobStartedAtKey   = "cicd.github.workflow.job.started_at"
	GitHubWorkflowJobCompletedAtKey = "cicd.github.workflow.job.completed_at"
	GitHubWorkflowJobStatusKey      = "cicd.github.workflow.job.status"
	GitHubWorkflowJobIDKey          = "cicd.github.workflow.job.id"
	GitHubWorkflowJobNameKey        = "cicd.github.workflow.job.name"
	GitHubWorkflowJobURLKey         = "cicd.github.workflow.job.url"

	SCMGitRepositoryNameKey        = "scm.git.repository.name"
	SCMGitRepositoryOwnerKey       = "scm.git.repository.owner"
	SCMGitRepositoryURLKey         = "scm.git.repository.url"
	SCMGitRepositoryIDKey          = "scm.git.repository.id"
	SCMGitHeadBranchKey            = "scm.git.head.branch"
	SCMGitHeadCommitSHAKey         = "scm.git.head.commit.sha"
	SCMGitHeadCommitMessageKey     = "scm.git.head.commit.message"
	SCMGitHeadCommitAuthorLoginKey = "scm.git.head.commit.author.login"
	SCMGitHeadCommitAuthorEmailKey = "scm.git.head.commit.author.email"
	SCMGitHeadCommitAuthorNameKey  = "scm.git.head.commit.author.name"

	SCMGitPullRequestsURLKey = "scm.git.pull_requests.url"
)

func eventToTraces(config *Config, event any) (ptrace.Traces, error) {
	var (
		traces        = ptrace.NewTraces()
		resourceSpans = traces.ResourceSpans().AppendEmpty()
		attrs         = resourceSpans.Resource().Attributes()
	)

	switch e := event.(type) {
	case *github.WorkflowJobEvent:
		// semconv
		serviceName := generateServiceName(config, e.GetRepo().GetFullName())

		attrs.PutStr(string(semconv.ServiceNameKey), serviceName)

		// traceID is the SHA of the head commit
		traceID, err := ids.TraceFromString(e.GetWorkflowJob().GetHeadSHA())
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("failed to generate trace ID: %w", err)
		}

		// spanID is the SHA of the head commit, the job ID, and the job attempt
		spanID, err := generateSpanID(e.GetWorkflowJob().GetHeadSHA(), strconv.FormatInt(e.GetWorkflowJob().GetID(), 10), strconv.FormatInt(e.GetWorkflowJob().GetRunAttempt(), 10))
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("failed to generate span ID: %w", err)
		}

		// parent spanID is the SHA of the head commit, the run ID, and the run attempt
		parentSpanID, err := generateSpanID(e.GetWorkflowJob().GetHeadSHA(), strconv.FormatInt(e.GetWorkflowJob().GetRunID(), 10), strconv.FormatInt(e.GetWorkflowJob().GetRunAttempt(), 10))
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("failed to generate span ID: %w", err)
		}

		var (
			scopeSpans = resourceSpans.ScopeSpans().AppendEmpty()
			span       = scopeSpans.Spans().AppendEmpty()
			spanAttrs  = span.Attributes()
		)

		spanAttrs.PutStr(string(semconv.CICDPipelineTaskNameKey), e.GetWorkflowJob().GetName())
		spanAttrs.PutStr(string(semconv.CICDPipelineTaskNameKey), e.GetWorkflowJob().GetName())
		spanAttrs.PutInt(string(semconv.CICDPipelineTaskRunIDKey), e.GetWorkflowJob().GetID())
		spanAttrs.PutStr(string(semconv.CICDPipelineTaskRunURLFullKey), e.GetWorkflowJob().GetHTMLURL())

		spanAttrs.PutStr(string(semconv.VCSRepositoryURLFullKey), e.GetRepo().GetHTMLURL())
		spanAttrs.PutStr(string(semconv.VCSRepositoryRefRevisionKey), e.GetWorkflowJob().GetHeadSHA())
		spanAttrs.PutStr(string(semconv.VCSRepositoryRefNameKey), e.GetWorkflowJob().GetHeadBranch())

		// custom attributes

		// cicd.github.workflow.job
		spanAttrs.PutStr(GitHubWorkflowJobConclusionKey, e.GetWorkflowJob().GetConclusion())
		spanAttrs.PutStr(GitHubWorkflowJobCreatedAtKey, e.GetWorkflowJob().GetCreatedAt().Format(time.RFC3339))
		spanAttrs.PutStr(GitHubWorkflowJobStartedAtKey, e.GetWorkflowJob().GetStartedAt().Format(time.RFC3339))
		spanAttrs.PutStr(GitHubWorkflowJobCompletedAtKey, e.GetWorkflowJob().GetCompletedAt().Format(time.RFC3339))
		spanAttrs.PutStr(GitHubWorkflowJobStatusKey, e.GetWorkflowJob().GetStatus())
		spanAttrs.PutStr(GitHubWorkflowJobURLKey, e.GetWorkflowJob().GetHTMLURL())
		spanAttrs.PutInt(GitHubWorkflowJobIDKey, e.GetWorkflowJob().GetID())
		spanAttrs.PutStr(GitHubWorkflowJobNameKey, e.GetWorkflowJob().GetName())

		// scm.git
		spanAttrs.PutStr(SCMGitRepositoryNameKey, e.GetRepo().GetName())
		spanAttrs.PutStr(SCMGitRepositoryOwnerKey, e.GetRepo().GetOwner().GetLogin())
		spanAttrs.PutStr(SCMGitRepositoryURLKey, e.GetRepo().GetHTMLURL())
		spanAttrs.PutInt(SCMGitRepositoryIDKey, e.GetRepo().GetID())
		spanAttrs.PutStr(SCMGitHeadBranchKey, e.GetWorkflowJob().GetHeadBranch())

		span.SetName(e.GetWorkflowJob().GetName())
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(e.GetWorkflowJob().GetStartedAt().Time))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(e.GetWorkflowJob().GetCompletedAt().Time))
		span.SetTraceID(traceID)
		span.SetSpanID(spanID)
		span.SetParentSpanID(parentSpanID)
		span.SetKind(ptrace.SpanKindServer)

	case *github.WorkflowRunEvent:
		// semconv
		serviceName := generateServiceName(config, e.GetRepo().GetFullName())

		attrs.PutStr(string(semconv.ServiceNameKey), serviceName)

		// traceID is the SHA of the head commit
		traceID, err := ids.TraceFromString(e.GetWorkflowRun().GetHeadSHA())
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("failed to generate trace ID: %w", err)
		}

		// spanID is the SHA of the head commit, the run ID, and the run attempt
		spanID, err := generateSpanID(e.GetWorkflowRun().GetHeadSHA(), strconv.FormatInt(e.GetWorkflowRun().GetID(), 10), strconv.Itoa(e.GetWorkflowRun().GetRunAttempt()))
		if err != nil {
			return ptrace.Traces{}, fmt.Errorf("failed to generate span ID: %w", err)
		}

		var (
			scopeSpans = resourceSpans.ScopeSpans().AppendEmpty()
			span       = scopeSpans.Spans().AppendEmpty()
			spanAttrs  = span.Attributes()
		)

		spanAttrs.PutStr(string(semconv.CICDPipelineNameKey), e.GetWorkflowRun().GetName())
		spanAttrs.PutInt(string(semconv.CICDPipelineRunIDKey), e.GetWorkflowRun().GetID())

		spanAttrs.PutStr(string(semconv.VCSRepositoryURLFullKey), e.GetRepo().GetHTMLURL())
		spanAttrs.PutStr(string(semconv.VCSRepositoryRefRevisionKey), e.GetWorkflowRun().GetHeadSHA())
		spanAttrs.PutStr(string(semconv.VCSRepositoryRefNameKey), e.GetWorkflowRun().GetHeadBranch())

		// custom attributes

		// cicd.github.workflow.run
		spanAttrs.PutStr(GitHubWorkflowRunConclusionKey, e.GetWorkflowRun().GetConclusion())
		spanAttrs.PutStr(GitHubWorkflowRunCreatedAtKey, e.GetWorkflowRun().GetCreatedAt().Format(time.RFC3339))
		spanAttrs.PutStr(GitHubWorkflowRunStartedAtKey, e.GetWorkflowRun().GetRunStartedAt().Format(time.RFC3339))
		spanAttrs.PutStr(GitHubWorkflowRunUpdatedAtKey, e.GetWorkflowRun().GetUpdatedAt().Format(time.RFC3339))
		spanAttrs.PutInt(GitHubWorkflowRunAttemptKey, int64(e.GetWorkflowRun().GetRunAttempt()))
		spanAttrs.PutStr(GitHubWorkflowRunStatusKey, e.GetWorkflowRun().GetStatus())
		spanAttrs.PutStr(GitHubWorkflowRunURLKey, e.GetWorkflowRun().GetHTMLURL())
		spanAttrs.PutInt(GitHubWorkflowRunIDKey, e.GetWorkflowRun().GetID())
		spanAttrs.PutStr(GitHubWorkflowRunNameKey, e.GetWorkflowRun().GetName())

		// scm.git
		spanAttrs.PutStr(SCMGitRepositoryNameKey, e.GetRepo().GetName())
		spanAttrs.PutStr(SCMGitRepositoryOwnerKey, e.GetRepo().GetOwner().GetLogin())
		spanAttrs.PutStr(SCMGitRepositoryURLKey, e.GetRepo().GetHTMLURL())
		spanAttrs.PutInt(SCMGitRepositoryIDKey, e.GetRepo().GetID())
		spanAttrs.PutStr(SCMGitHeadBranchKey, e.GetWorkflowRun().GetHeadBranch())
		spanAttrs.PutStr(SCMGitHeadCommitSHAKey, e.GetWorkflowRun().GetHeadSHA())
		spanAttrs.PutStr(SCMGitHeadCommitMessageKey, e.GetWorkflowRun().GetHeadCommit().GetMessage())
		spanAttrs.PutStr(SCMGitHeadCommitAuthorLoginKey, e.GetWorkflowRun().GetHeadCommit().GetAuthor().GetLogin())
		spanAttrs.PutStr(SCMGitHeadCommitAuthorEmailKey, e.GetWorkflowRun().GetHeadCommit().GetAuthor().GetEmail())
		spanAttrs.PutStr(SCMGitHeadCommitAuthorNameKey, e.GetWorkflowRun().GetHeadCommit().GetAuthor().GetName())

		if len(e.GetWorkflowRun().PullRequests) > 0 {
			var prUrls []string
			for _, pr := range e.GetWorkflowRun().PullRequests {
				prUrls = append(prUrls, cleanGitHubURL(pr.GetURL()))
			}
			spanAttrs.PutStr(SCMGitPullRequestsURLKey, strings.Join(prUrls, ";"))
		}

		span.SetName(e.GetWorkflowRun().GetName())
		span.SetStartTimestamp(pcommon.NewTimestampFromTime(e.GetWorkflowRun().GetRunStartedAt().Time))
		span.SetEndTimestamp(pcommon.NewTimestampFromTime(e.GetWorkflowRun().GetUpdatedAt().Time))
		span.SetTraceID(traceID)
		span.SetSpanID(spanID)
		span.SetKind(ptrace.SpanKindServer)

		switch e.GetWorkflowRun().GetConclusion() {
		case "success":
			span.Status().SetCode(ptrace.StatusCodeOk)
		case "failure":
			span.Status().SetCode(ptrace.StatusCodeError)
		default:
			span.Status().SetCode(ptrace.StatusCodeUnset)
		}

		span.Status().SetMessage(e.GetWorkflowRun().GetConclusion())
	}

	return traces, nil
}

func generateServiceName(config *Config, fullName string) string {
	if config.CustomServiceName != "" {
		return config.CustomServiceName
	}
	formattedName := strings.ToLower(strings.ReplaceAll(strings.ReplaceAll(fullName, "/", "-"), "_", "-"))
	return fmt.Sprintf("%s%s%s", config.ServiceNamePrefix, formattedName, config.ServiceNameSuffix)
}

func generateSpanID(parts ...string) (pcommon.SpanID, error) {
	var (
		input     = strings.Join(parts, "")
		hash      = sha256.Sum256([]byte(fmt.Sprintf("%ss", input)))
		spanIDHex = hex.EncodeToString(hash[:])
		spanID    pcommon.SpanID
	)

	_, err := hex.Decode(spanID[:], []byte(spanIDHex[16:32]))
	if err != nil {
		return pcommon.SpanID{}, err
	}

	return spanID, nil
}

func cleanGitHubURL(apiURL string) string {
	return strings.Replace(apiURL, "api.", "", 1)
}
