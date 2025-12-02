package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type SecurityScanRequest struct {
	RepositoryURL string
	Branch        string
	CommitSHA     string
	ScanTypes     []string // "sast", "dast", "dependency", "secrets"
}

type SecurityScanResult struct {
	ScanID          string
	Status          string
	Vulnerabilities []Vulnerability
	CompletedAt     time.Time
	ReportURL       string
}

type Vulnerability struct {
	ID          string
	Severity    string // "critical", "high", "medium", "low"
	Title       string
	Description string
	FilePath    string
	LineNumber  int
	Remediation string
}

type AgentContext struct {
	AgentID     string
	SessionID   string
	Permissions []string
}

// SecurityScanWorkflow orchestrates comprehensive security scanning for code repositories.
// This workflow is designed to be called by AI coding agents to validate code changes.
//
// Integration points:
//   - Called by: AgentOrchestrator.ValidateChanges()
//   - Uses: SecurityScanner service (//services/scanner)
//   - Reports to: ComplianceReporter (//services/compliance)
func SecurityScanWorkflow(ctx workflow.Context, request SecurityScanRequest, agentCtx AgentContext) (*SecurityScanResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting security scan workflow",
		"repo", request.RepositoryURL,
		"commit", request.CommitSHA,
		"agentID", agentCtx.AgentID)

	// Validate agent has required permissions
	if !hasPermission(agentCtx.Permissions, "security:scan:execute") {
		logger.Warn("Agent lacks required permissions", "agentID", agentCtx.AgentID)
		return &SecurityScanResult{
			Status: "PERMISSION_DENIED",
		}, nil
	}

	// Configure retry policy for scanning activities
	// Security scans are expensive - limit retries
	scanOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 30,
		HeartbeatTimeout:    time.Minute * 2,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second * 5,
			BackoffCoefficient: 1.5,
			MaximumInterval:    time.Minute * 2,
			MaximumAttempts:    2, // Limit retries for expensive operations
		},
	}
	ctx = workflow.WithActivityOptions(ctx, scanOptions)

	var allVulnerabilities []Vulnerability

	// Run scan types in parallel for efficiency
	// TODO: Add rate limiting for API-bound scanners
	futures := make(map[string]workflow.Future)
	for _, scanType := range request.ScanTypes {
		switch scanType {
		case "sast":
			futures["sast"] = workflow.ExecuteActivity(ctx, RunSASTScan, request)
		case "dast":
			futures["dast"] = workflow.ExecuteActivity(ctx, RunDASTScan, request)
		case "dependency":
			futures["dependency"] = workflow.ExecuteActivity(ctx, RunDependencyScan, request)
		case "secrets":
			futures["secrets"] = workflow.ExecuteActivity(ctx, RunSecretsScan, request)
		}
	}

	// Collect results
	for scanType, future := range futures {
		var scanResult ScanTypeResult
		if err := future.Get(ctx, &scanResult); err != nil {
			logger.Error("Scan failed", "type", scanType, "error", err)
			continue
		}
		allVulnerabilities = append(allVulnerabilities, scanResult.Vulnerabilities...)
	}

	// Generate report
	var reportResult ReportResult
	reportOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			MaximumAttempts: 3,
		},
	}
	reportCtx := workflow.WithActivityOptions(ctx, reportOptions)
	err := workflow.ExecuteActivity(reportCtx, GenerateSecurityReport, allVulnerabilities).Get(ctx, &reportResult)
	if err != nil {
		logger.Error("Report generation failed", "error", err)
	}

	// Notify compliance service for critical vulnerabilities
	criticalCount := countBySeverity(allVulnerabilities, "critical")
	if criticalCount > 0 {
		workflow.ExecuteActivity(ctx, NotifyComplianceTeam, NotificationRequest{
			Type:    "CRITICAL_VULNERABILITIES",
			Count:   criticalCount,
			ScanID:  reportResult.ReportID,
			AgentID: agentCtx.AgentID,
		})
	}

	return &SecurityScanResult{
		ScanID:          reportResult.ReportID,
		Status:          determineStatus(allVulnerabilities),
		Vulnerabilities: allVulnerabilities,
		CompletedAt:     workflow.Now(ctx),
		ReportURL:       reportResult.URL,
	}, nil
}

func hasPermission(permissions []string, required string) bool {
	for _, p := range permissions {
		if p == required || p == "security:*" {
			return true
		}
	}
	return false
}

func countBySeverity(vulns []Vulnerability, severity string) int {
	count := 0
	for _, v := range vulns {
		if v.Severity == severity {
			count++
		}
	}
	return count
}

func determineStatus(vulns []Vulnerability) string {
	for _, v := range vulns {
		if v.Severity == "critical" {
			return "FAILED_CRITICAL"
		}
	}
	for _, v := range vulns {
		if v.Severity == "high" {
			return "FAILED_HIGH"
		}
	}
	if len(vulns) > 0 {
		return "PASSED_WITH_WARNINGS"
	}
	return "PASSED"
}
