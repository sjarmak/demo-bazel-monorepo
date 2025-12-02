package workflows

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	OrderTaskQueue    = "order-processing"
	PaymentTaskQueue  = "payment-processing"
	SecurityTaskQueue = "security-scanning"
)

// WorkerConfig holds configuration for Temporal workers
type WorkerConfig struct {
	TemporalHost      string
	TemporalNamespace string
	WorkerID          string
}

// StartOrderWorker initializes and starts the order processing worker
func StartOrderWorker(config WorkerConfig) error {
	c, err := client.Dial(client.Options{
		HostPort:  config.TemporalHost,
		Namespace: config.TemporalNamespace,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	w := worker.New(c, OrderTaskQueue, worker.Options{
		Identity: config.WorkerID,
	})

	// Register workflows
	w.RegisterWorkflow(OrderWorkflow)

	// Register activities
	w.RegisterActivity(ValidateInventory)
	w.RegisterActivity(GenerateShippingLabel)
	w.RegisterActivity(RefundPayment)

	log.Printf("Starting order worker on queue: %s", OrderTaskQueue)
	return w.Run(worker.InterruptCh())
}

// StartPaymentWorker initializes and starts the payment processing worker
func StartPaymentWorker(config WorkerConfig) error {
	c, err := client.Dial(client.Options{
		HostPort:  config.TemporalHost,
		Namespace: config.TemporalNamespace,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	w := worker.New(c, PaymentTaskQueue, worker.Options{
		Identity: config.WorkerID,
	})

	// Register both v1 and v2 workflows for migration period
	w.RegisterWorkflow(PaymentWorkflow)
	w.RegisterWorkflow(PaymentWorkflowV2)

	// Register activities
	w.RegisterActivity(CheckFraud)
	w.RegisterActivity(CheckFraudV2)
	w.RegisterActivity(ValidateCard)
	w.RegisterActivity(ChargePaymentMethod)
	w.RegisterActivity(ChargePaymentMethodV2)
	w.RegisterActivity(SendPaymentConfirmation)

	log.Printf("Starting payment worker on queue: %s", PaymentTaskQueue)
	return w.Run(worker.InterruptCh())
}

// StartSecurityWorker initializes and starts the security scanning worker
// This worker handles AI agent-initiated security scans
func StartSecurityWorker(config WorkerConfig) error {
	c, err := client.Dial(client.Options{
		HostPort:  config.TemporalHost,
		Namespace: config.TemporalNamespace,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	w := worker.New(c, SecurityTaskQueue, worker.Options{
		Identity:                  config.WorkerID,
		MaxConcurrentActivityExecutionSize: 5, // Limit concurrent scans
	})

	// Register security workflow
	w.RegisterWorkflow(SecurityScanWorkflow)

	// Register scan activities
	w.RegisterActivity(RunSASTScan)
	w.RegisterActivity(RunDASTScan)
	w.RegisterActivity(RunDependencyScan)
	w.RegisterActivity(RunSecretsScan)
	w.RegisterActivity(GenerateSecurityReport)
	w.RegisterActivity(NotifyComplianceTeam)

	log.Printf("Starting security worker on queue: %s", SecurityTaskQueue)
	return w.Run(worker.InterruptCh())
}
