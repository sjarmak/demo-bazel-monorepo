package workflows

import (
	"testing"

	"go.temporal.io/sdk/testsuite"
)

func TestOrderWorkflow_Success(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	// Mock activities
	env.OnActivity(ValidateInventory, []OrderItem{}).Return(&InventoryResult{Available: true}, nil)
	env.OnActivity(GenerateShippingLabel, "order-123").Return(&ShippingResult{TrackingNumber: "TRK-123"}, nil)

	// Mock child workflow
	env.OnWorkflow(PaymentWorkflow, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     99.99,
	}).Return(&PaymentResult{
		TransactionID: "txn-789",
		Status:        "APPROVED",
	}, nil)

	request := OrderRequest{
		OrderID:     "order-123",
		CustomerID:  "customer-456",
		Items:       []OrderItem{},
		TotalAmount: 99.99,
	}

	env.ExecuteWorkflow(OrderWorkflow, request)

	var result OrderResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "COMPLETED" {
		t.Errorf("Expected status COMPLETED, got %s", result.Status)
	}

	if result.PaymentID != "txn-789" {
		t.Errorf("Expected payment ID txn-789, got %s", result.PaymentID)
	}
}

func TestOrderWorkflow_InventoryUnavailable(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(ValidateInventory, []OrderItem{}).Return(&InventoryResult{Available: false}, nil)

	request := OrderRequest{
		OrderID:     "order-123",
		CustomerID:  "customer-456",
		Items:       []OrderItem{},
		TotalAmount: 99.99,
	}

	env.ExecuteWorkflow(OrderWorkflow, request)

	var result OrderResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "INVENTORY_UNAVAILABLE" {
		t.Errorf("Expected status INVENTORY_UNAVAILABLE, got %s", result.Status)
	}
}
