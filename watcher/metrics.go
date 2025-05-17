package main

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// Buffer metrics
	activeBufferSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "event_processor_active_buffer_size",
		Help: "Current number of events in the active buffer",
	})

	backupBufferSize = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "event_processor_backup_buffer_size",
		Help: "Current number of events in the backup buffer",
	})

	// Event processing metrics
	eventsProcessedTotal = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "event_processor_events_processed_total",
		Help: "Total number of events processed, labeled by operation type and status",
	}, []string{"operation_type", "status"})

	eventProcessingDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "event_processor_processing_duration_seconds",
		Help:    "Time taken to process events",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation_type"})

	// Batch processing metrics
	batchSize = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "event_processor_batch_size",
		Help:    "Size of processed batches",
		Buckets: []float64{1, 2, 5, 10, 20, 50, 100},
	})

	batchProcessingDuration = promauto.NewHistogram(prometheus.HistogramOpts{
		Name:    "event_processor_batch_processing_duration_seconds",
		Help:    "Time taken to process a batch of events",
		Buckets: prometheus.DefBuckets,
	})

	// Error metrics
	processingErrors = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "event_processor_errors_total",
		Help: "Total number of processing errors, labeled by error type",
	}, []string{"error_type"})

	// Queue metrics
	queueLength = promauto.NewGauge(prometheus.GaugeOpts{
		Name: "event_processor_queue_length",
		Help: "Current number of events in the processing queue",
	})

	// Typesense operation metrics
	typesenseOperationDuration = promauto.NewHistogramVec(prometheus.HistogramOpts{
		Name:    "event_processor_typesense_operation_duration_seconds",
		Help:    "Time taken for Typesense operations",
		Buckets: prometheus.DefBuckets,
	}, []string{"operation"})

	// Rate limiting metrics
	rateLimitHits = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "event_processor_rate_limit_hits_total",
		Help: "Total number of rate limit hits, labeled by operation type",
	}, []string{"operation_type"})
)

// MetricsCollector provides methods to update metrics
type MetricsCollector struct{}

// NewMetricsCollector creates a new metrics collector
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{}
}

// UpdateBufferSizes updates the buffer size metrics
func (m *MetricsCollector) UpdateBufferSizes(activeSize, backupSize int) {
	activeBufferSize.Set(float64(activeSize))
	backupBufferSize.Set(float64(backupSize))
}

// RecordEventProcessed records a processed event
func (m *MetricsCollector) RecordEventProcessed(operationType, status string) {
	eventsProcessedTotal.WithLabelValues(operationType, status).Inc()
}

// RecordEventProcessingDuration records the time taken to process an event
func (m *MetricsCollector) RecordEventProcessingDuration(operationType string, duration float64) {
	eventProcessingDuration.WithLabelValues(operationType).Observe(duration)
}

// RecordBatchSize records the size of a processed batch
func (m *MetricsCollector) RecordBatchSize(size int) {
	batchSize.Observe(float64(size))
}

// RecordBatchProcessingDuration records the time taken to process a batch
func (m *MetricsCollector) RecordBatchProcessingDuration(duration float64) {
	batchProcessingDuration.Observe(duration)
}

// RecordError records a processing error
func (m *MetricsCollector) RecordError(errorType string) {
	processingErrors.WithLabelValues(errorType).Inc()
}

// UpdateQueueLength updates the queue length metric
func (m *MetricsCollector) UpdateQueueLength(length int) {
	queueLength.Set(float64(length))
}

// RecordTypesenseOperationDuration records the time taken for a Typesense operation
func (m *MetricsCollector) RecordTypesenseOperationDuration(operation string, duration float64) {
	typesenseOperationDuration.WithLabelValues(operation).Observe(duration)
}

// RecordRateLimit records a rate limit hit for an operation type
func (m *MetricsCollector) RecordRateLimit(operationType string) {
	rateLimitHits.WithLabelValues(operationType).Inc()
}
