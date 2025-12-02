package workflows

import (
	"testing"
	"time"

	"go.temporal.io/sdk/testsuite"
)

func TestSecurityScanWorkflow_PassedClean(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	request := SecurityScanRequest{
		RepositoryURL: "https://github.com/example/repo",
		Branch:        "main",
		CommitSHA:     "abc123",
		ScanTypes:     []string{"sast", "secrets"},
	}

	agentCtx := AgentContext{
		AgentID:     "agent-001",
		SessionID:   "session-xyz",
		Permissions: []string{"security:scan:execute"},
	}

	env.OnActivity(RunSASTScan, request).Return(&ScanTypeResult{
		ScanType:        "sast",
		Vulnerabilities: []Vulnerability{},
		Duration:        time.Minute * 5,
	}, nil)

	env.OnActivity(RunSecretsScan, request).Return(&ScanTypeResult{
		ScanType:        "secrets",
		Vulnerabilities: []Vulnerability{},
		Duration:        time.Minute * 1,
	}, nil)

	env.OnActivity(GenerateSecurityReport, []Vulnerability{}).Return(&ReportResult{
		ReportID: "SEC-123",
		URL:      "https://security.example.com/reports/SEC-123",
	}, nil)

	env.ExecuteWorkflow(SecurityScanWorkflow, request, agentCtx)

	var result SecurityScanResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "PASSED" {
		t.Errorf("Expected status PASSED, got %s", result.Status)
	}

	if len(result.Vulnerabilities) != 0 {
		t.Errorf("Expected 0 vulnerabilities, got %d", len(result.Vulnerabilities))
	}
}

func TestSecurityScanWorkflow_PermissionDenied(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	request := SecurityScanRequest{
		RepositoryURL: "https://github.com/example/repo",
		Branch:        "main",
		CommitSHA:     "abc123",
		ScanTypes:     []string{"sast"},
	}

	agentCtx := AgentContext{
		AgentID:     "agent-001",
		SessionID:   "session-xyz",
		Permissions: []string{"read:only"}, // Missing security permissions
	}

	env.ExecuteWorkflow(SecurityScanWorkflow, request, agentCtx)

	var result SecurityScanResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "PERMISSION_DENIED" {
		t.Errorf("Expected status PERMISSION_DENIED, got %s", result.Status)
	}
}

func TestSecurityScanWorkflow_CriticalVulnerabilities(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	request := SecurityScanRequest{
		RepositoryURL: "https://github.com/example/repo",
		Branch:        "main",
		CommitSHA:     "abc123",
		ScanTypes:     []string{"dependency"},
	}

	agentCtx := AgentContext{
		AgentID:     "agent-001",
		SessionID:   "session-xyz",
		Permissions: []string{"security:*"},
	}

	criticalVuln := Vulnerability{
		ID:       "CVE-2024-99999",
		Severity: "critical",
		Title:    "Remote Code Execution in core library",
		FilePath: "go.mod",
	}

	env.OnActivity(RunDependencyScan, request).Return(&ScanTypeResult{
		ScanType:        "dependency",
		Vulnerabilities: []Vulnerability{criticalVuln},
		Duration:        time.Minute * 2,
	}, nil)

	env.OnActivity(GenerateSecurityReport, []Vulnerability{criticalVuln}).Return(&ReportResult{
		ReportID: "SEC-456",
		URL:      "https://security.example.com/reports/SEC-456",
	}, nil)

	env.OnActivity(NotifyComplianceTeam, NotificationRequest{
		Type:    "CRITICAL_VULNERABILITIES",
		Count:   1,
		ScanID:  "SEC-456",
		AgentID: "agent-001",
	}).Return(nil)

	env.ExecuteWorkflow(SecurityScanWorkflow, request, agentCtx)

	var result SecurityScanResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "FAILED_CRITICAL" {
		t.Errorf("Expected status FAILED_CRITICAL, got %s", result.Status)
	}

	if len(result.Vulnerabilities) != 1 {
		t.Errorf("Expected 1 vulnerability, got %d", len(result.Vulnerabilities))
	}
}
