# Temporal Workflows Architecture

> **Last updated:** February 2024

This document describes the Temporal workflow architecture used in the Antilibrary platform.

## Overview

We use [Temporal](https://temporal.io) for durable workflow orchestration across our microservices.

## Workflow Inventory

| Workflow | Task Queue | Description |
|----------|------------|-------------|
| `OrderWorkflow` | `order-processing` | End-to-end order fulfillment |
| `PaymentWorkflow` | `payment-processing` | Payment processing with fraud detection |
| `SecurityScanWorkflow` | `security-scanning` | AI agent-initiated security scans |

## Retry Policies

### Standard Configuration

All workflows use a standard retry policy unless otherwise specified:

```go
RetryPolicy: &temporal.RetryPolicy{
    InitialInterval:    time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    time.Minute,
    MaximumAttempts:    3,
}
```

### Payment-Specific Retries

Payment workflows use enhanced retry logic:
- Initial interval: 1 second
- Maximum attempts: 3
- Non-retryable errors: `FraudDetectedError`, `InsufficientFundsError`

**IMPORTANT:** Payment retries must be idempotent to prevent duplicate charges.

## Task Queues

### Worker Configuration

Each task queue has dedicated workers:

```go
// Order processing
StartOrderWorker(WorkerConfig{
    TemporalHost:      "temporal:7233",
    TemporalNamespace: "default",
    WorkerID:          "order-worker-1",
})
```

### Scaling Considerations

- Order workers: 3 replicas recommended
- Payment workers: 2 replicas with circuit breaker
- Security workers: 5 concurrent activity limit

## Activity Patterns

### Fire-and-Forget

For non-critical activities like notifications:

```go
workflow.ExecuteActivity(ctx, SendConfirmation, data)
// Note: We don't wait for the result
```

### Child Workflows

Payment is executed as a child workflow from orders:

```go
childOptions := workflow.ChildWorkflowOptions{
    WorkflowID: "payment-" + orderID,
}
childCtx := workflow.WithChildOptions(ctx, childOptions)
err := workflow.ExecuteChildWorkflow(childCtx, PaymentWorkflow, request).Get(ctx, &result)
```

## Monitoring

### Metrics

Key metrics to monitor:
- `temporal_workflow_started_total`
- `temporal_workflow_completed_total`
- `temporal_activity_execution_latency`

### Dashboards

Grafana dashboards are available at: `https://grafana.example.com/temporal`

## Migration Notes

### From v1 to v2

The `PaymentWorkflow` was updated in v2:
- Added `CheckFraudV2` activity
- Changed retry policy (MaxAttempts: 5 → 3)
- Added `ValidateCard` parallel activity

See `//workflows/payment_workflow.go` for the new implementation.

## References

- [Temporal Go SDK Documentation](https://docs.temporal.io/go)
- Internal design doc: `go/temporal-migration`
