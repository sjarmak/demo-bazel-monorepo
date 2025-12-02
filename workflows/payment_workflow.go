package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type PaymentRequest struct {
	OrderID    string
	CustomerID string
	Amount     float64
	Currency   string
}

type PaymentResult struct {
	TransactionID string
	Status        string
	ProcessedAt   time.Time
	ErrorMessage  string
}

// PaymentWorkflow handles payment processing with fraud detection.
//
// DEPRECATED: Use PaymentWorkflowV2 for new integrations.
// This workflow will be removed in v3.0.
//
// Retry Policy Configuration:
//   - InitialInterval: 2 seconds (NOTE: differs from OrderWorkflow's 1 second)
//   - BackoffCoefficient: 2.0
//   - MaximumInterval: 30 seconds
//   - MaximumAttempts: 5
func PaymentWorkflow(ctx workflow.Context, request PaymentRequest) (*PaymentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting payment workflow", "orderID", request.OrderID, "amount", request.Amount)

	// Activity options with specific retry policy for payment operations
	// WARNING: MaximumAttempts of 5 may cause duplicate charges if not idempotent
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 2,
		HeartbeatTimeout:    time.Second * 30,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second * 2,
			BackoffCoefficient:     2.0,
			MaximumInterval:        time.Second * 30,
			MaximumAttempts:        5,
			NonRetryableErrorTypes: []string{"FraudDetectedError", "InsufficientFundsError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Run fraud detection
	var fraudResult FraudCheckResult
	err := workflow.ExecuteActivity(ctx, CheckFraud, request).Get(ctx, &fraudResult)
	if err != nil {
		return &PaymentResult{
			Status:       "FRAUD_CHECK_FAILED",
			ErrorMessage: err.Error(),
		}, nil
	}

	if fraudResult.RiskScore > 0.8 {
		logger.Warn("High fraud risk detected", "score", fraudResult.RiskScore)
		return &PaymentResult{
			Status:       "FRAUD_SUSPECTED",
			ErrorMessage: fmt.Sprintf("Risk score %.2f exceeds threshold", fraudResult.RiskScore),
		}, nil
	}

	// Step 2: Charge payment method
	var chargeResult ChargeResult
	err = workflow.ExecuteActivity(ctx, ChargePaymentMethod, request).Get(ctx, &chargeResult)
	if err != nil {
		logger.Error("Payment charge failed", "error", err)
		return &PaymentResult{
			Status:       "CHARGE_FAILED",
			ErrorMessage: err.Error(),
		}, nil
	}

	// Step 3: Send confirmation (fire and forget)
	workflow.ExecuteActivity(ctx, SendPaymentConfirmation, chargeResult.TransactionID)

	return &PaymentResult{
		TransactionID: chargeResult.TransactionID,
		Status:        "APPROVED",
		ProcessedAt:   workflow.Now(ctx),
	}, nil
}

// PaymentWorkflowV2 is the updated payment workflow with improved retry logic.
// Uses circuit breaker pattern for external payment gateway calls.
func PaymentWorkflowV2(ctx workflow.Context, request PaymentRequest) (*PaymentResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting payment workflow v2", "orderID", request.OrderID)

	// Updated retry policy with circuit breaker behavior
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 3,
		HeartbeatTimeout:    time.Second * 45,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:        time.Second,
			BackoffCoefficient:     1.5,
			MaximumInterval:        time.Second * 15,
			MaximumAttempts:        3,
			NonRetryableErrorTypes: []string{"FraudDetectedError", "InsufficientFundsError", "InvalidCardError"},
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Parallel fraud check and card validation
	var fraudResult FraudCheckResult
	var cardValid bool

	selector := workflow.NewSelector(ctx)

	fraudFuture := workflow.ExecuteActivity(ctx, CheckFraudV2, request)
	selector.AddFuture(fraudFuture, func(f workflow.Future) {
		f.Get(ctx, &fraudResult)
	})

	cardFuture := workflow.ExecuteActivity(ctx, ValidateCard, request.CustomerID)
	selector.AddFuture(cardFuture, func(f workflow.Future) {
		f.Get(ctx, &cardValid)
	})

	for i := 0; i < 2; i++ {
		selector.Select(ctx)
	}

	if !cardValid || fraudResult.RiskScore > 0.75 {
		return &PaymentResult{
			Status: "DECLINED",
		}, nil
	}

	var chargeResult ChargeResult
	err := workflow.ExecuteActivity(ctx, ChargePaymentMethodV2, request).Get(ctx, &chargeResult)
	if err != nil {
		return nil, err
	}

	return &PaymentResult{
		TransactionID: chargeResult.TransactionID,
		Status:        "APPROVED",
		ProcessedAt:   workflow.Now(ctx),
	}, nil
}
