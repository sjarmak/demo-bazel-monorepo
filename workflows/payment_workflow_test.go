package workflows

import (
	"testing"

	"go.temporal.io/sdk/testsuite"
)

func TestPaymentWorkflow_Approved(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(CheckFraud, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     50.00,
	}).Return(&FraudCheckResult{RiskScore: 0.1}, nil)

	env.OnActivity(ChargePaymentMethod, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     50.00,
	}).Return(&ChargeResult{TransactionID: "txn-abc"}, nil)

	env.OnActivity(SendPaymentConfirmation, "txn-abc").Return(nil)

	request := PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     50.00,
	}

	env.ExecuteWorkflow(PaymentWorkflow, request)

	var result PaymentResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "APPROVED" {
		t.Errorf("Expected status APPROVED, got %s", result.Status)
	}
}

func TestPaymentWorkflow_FraudDetected(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(CheckFraud, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     50.00,
	}).Return(&FraudCheckResult{RiskScore: 0.95}, nil)

	request := PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     50.00,
	}

	env.ExecuteWorkflow(PaymentWorkflow, request)

	var result PaymentResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "FRAUD_SUSPECTED" {
		t.Errorf("Expected status FRAUD_SUSPECTED, got %s", result.Status)
	}
}

func TestPaymentWorkflowV2_Approved(t *testing.T) {
	testSuite := &testsuite.WorkflowTestSuite{}
	env := testSuite.NewTestWorkflowEnvironment()

	env.OnActivity(CheckFraudV2, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     75.00,
	}).Return(&FraudCheckResult{RiskScore: 0.2}, nil)

	env.OnActivity(ValidateCard, "customer-456").Return(true, nil)

	env.OnActivity(ChargePaymentMethodV2, PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     75.00,
	}).Return(&ChargeResult{TransactionID: "txn-v2-123"}, nil)

	request := PaymentRequest{
		OrderID:    "order-123",
		CustomerID: "customer-456",
		Amount:     75.00,
	}

	env.ExecuteWorkflow(PaymentWorkflowV2, request)

	var result PaymentResult
	err := env.GetWorkflowResult(&result)

	if err != nil {
		t.Fatalf("Workflow failed: %v", err)
	}

	if result.Status != "APPROVED" {
		t.Errorf("Expected status APPROVED, got %s", result.Status)
	}
}
