package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

type OrderRequest struct {
	OrderID     string
	CustomerID  string
	Items       []OrderItem
	TotalAmount float64
}

type OrderItem struct {
	BookID   string
	Title    string
	Quantity int
	Price    float64
}

type OrderResult struct {
	OrderID       string
	Status        string
	PaymentID     string
	ShippingLabel string
	CompletedAt   time.Time
}

// OrderWorkflow orchestrates the complete order fulfillment process
// including inventory check, payment processing, and shipping.
//
// Retry Policy: 3 attempts with exponential backoff starting at 1 second.
// This workflow calls: ValidateInventory, ProcessPayment, GenerateShippingLabel
func OrderWorkflow(ctx workflow.Context, request OrderRequest) (*OrderResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting order workflow", "orderID", request.OrderID)

	// Configure activity options with retry policy
	// NOTE: RetryPolicy backoff coefficient should match PaymentWorkflow
	activityOptions := workflow.ActivityOptions{
		StartToCloseTimeout: time.Minute * 5,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, activityOptions)

	// Step 1: Validate inventory availability
	var inventoryResult InventoryResult
	err := workflow.ExecuteActivity(ctx, ValidateInventory, request.Items).Get(ctx, &inventoryResult)
	if err != nil {
		logger.Error("Inventory validation failed", "error", err)
		return nil, err
	}

	if !inventoryResult.Available {
		return &OrderResult{
			OrderID: request.OrderID,
			Status:  "INVENTORY_UNAVAILABLE",
		}, nil
	}

	// Step 2: Process payment via child workflow
	childOptions := workflow.ChildWorkflowOptions{
		WorkflowID: "payment-" + request.OrderID,
	}
	childCtx := workflow.WithChildOptions(ctx, childOptions)

	paymentRequest := PaymentRequest{
		OrderID:    request.OrderID,
		CustomerID: request.CustomerID,
		Amount:     request.TotalAmount,
	}

	var paymentResult PaymentResult
	err = workflow.ExecuteChildWorkflow(childCtx, PaymentWorkflow, paymentRequest).Get(ctx, &paymentResult)
	if err != nil {
		logger.Error("Payment processing failed", "error", err)
		return nil, err
	}

	if paymentResult.Status != "APPROVED" {
		return &OrderResult{
			OrderID:   request.OrderID,
			Status:    "PAYMENT_DECLINED",
			PaymentID: paymentResult.TransactionID,
		}, nil
	}

	// Step 3: Generate shipping label
	var shippingResult ShippingResult
	err = workflow.ExecuteActivity(ctx, GenerateShippingLabel, request.OrderID).Get(ctx, &shippingResult)
	if err != nil {
		logger.Error("Shipping label generation failed", "error", err)
		// Compensate: refund payment
		_ = workflow.ExecuteActivity(ctx, RefundPayment, paymentResult.TransactionID).Get(ctx, nil)
		return nil, err
	}

	return &OrderResult{
		OrderID:       request.OrderID,
		Status:        "COMPLETED",
		PaymentID:     paymentResult.TransactionID,
		ShippingLabel: shippingResult.TrackingNumber,
		CompletedAt:   workflow.Now(ctx),
	}, nil
}
