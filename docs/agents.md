# AI Agent Integration Guide

> **Last updated:** March 2024

This document describes how to integrate AI coding agents with the Antilibrary monorepo.

## Overview

The Antilibrary platform supports AI coding agents through the Agent Integration Layer.
Agents can request security scans, execute workflows, and access code context.

## Getting Started

### Agent Registration

Register your agent using the `RegisterAgent` function in `//services/agents`:

```go
agent := agents.RegisterAgent(AgentConfig{
    Name:        "my-agent",
    Permissions: []string{"read", "write"},
})
```

**Note:** The `agents` package is located at `//services/agents/registry.go`.

### Permissions Model

Agents must have appropriate permissions to execute actions:

| Permission | Description |
|------------|-------------|
| `read` | Read-only access to code |
| `write` | Can create and modify files |
| `execute` | Can run workflows |
| `security` | Can initiate security scans |

## Workflow Integration

### Payment Processing

Agents can trigger payment workflows using the `PaymentWorkflow` function:

```go
result, err := temporal.ExecuteWorkflow(ctx, PaymentWorkflow, request)
```

**Configuration:**
- Retry attempts: 3
- Initial interval: 1 second  
- Backoff coefficient: 2.0

See `//workflows/payment_workflow.go` for implementation details.

### Security Scanning

For code validation, use `SecurityScanWorkflow`:

```go
request := SecurityScanRequest{
    RepositoryURL: "https://github.com/example/repo",
    ScanTypes:     []string{"sast", "dast"},
}
result, err := temporal.ExecuteWorkflow(ctx, SecurityScanWorkflow, request, agentCtx)
```

## API Reference

### Agent Context

All agent operations require an `AgentContext`:

```go
type AgentContext struct {
    AgentID     string
    SessionID   string
    Permissions []string
}
```

### Deprecated Functions

The following functions are deprecated and will be removed:

- `LegacyRegisterAgent()` - Use `RegisterAgent()` instead
- `CheckFraud()` - Use `CheckFraudV2()` instead

## Troubleshooting

### Common Issues

1. **Permission Denied errors**
   - Ensure your agent has the required permissions
   - Check that the session is still valid

2. **Workflow timeouts**
   - Default timeout is 5 minutes
   - For long-running scans, increase the timeout

### Support

Contact the Platform team on Slack: #platform-support

## Changelog

### v2.1.0 (March 2024)
- Added security scanning workflow
- Updated permissions model

### v2.0.0 (January 2024)  
- Migrated to Temporal from custom scheduler
- New agent registration API

### v1.0.0 (October 2023)
- Initial release
