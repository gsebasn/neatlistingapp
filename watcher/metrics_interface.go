package main

// MetricsCollector defines the interface for collecting and querying metrics
type MetricsCollector interface {
	// Update methods
	UpdateBufferSizes(activeSize, backupSize int)
	RecordEventProcessed(operationType, status string)
	RecordEventProcessingDuration(operationType string, duration float64)
	RecordBatchSize(size int)
	RecordBatchProcessingDuration(duration float64)
	RecordError(errorType string)
	UpdateQueueLength(length int)
	RecordTypesenseOperationDuration(operation string, duration float64)
	RecordRateLimit(operationType string)

	// Query methods
	getOperationCount(operationType, status string) float64
	getAverageProcessingDuration(operationType string) float64
}
