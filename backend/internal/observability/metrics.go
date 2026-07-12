package observability

import (
	"strconv"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	HTTPRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_http_requests_total",
			Help: "Total number of HTTP requests handled by the backend API.",
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autoinspect_http_request_duration_seconds",
			Help:    "HTTP request duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"method", "path", "status"},
	)

	HTTPRequestsInFlight = prometheus.NewGauge(
		prometheus.GaugeOpts{
			Name: "autoinspect_http_requests_in_flight",
			Help: "Number of HTTP requests currently being handled.",
		},
	)

	AnalysisJobsSubmittedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "autoinspect_analysis_jobs_submitted_total",
			Help: "Total number of analysis jobs submitted.",
		},
	)

	AnalysisImagesUploadedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "autoinspect_analysis_images_uploaded_total",
			Help: "Total number of images uploaded for analysis jobs.",
		},
	)

	AnalysisResultsReceivedTotal = prometheus.NewCounter(
		prometheus.CounterOpts{
			Name: "autoinspect_analysis_results_received_total",
			Help: "Total number of ML analysis result messages received.",
		},
	)

	AnalysisResultsHandledTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_analysis_results_handled_total",
			Help: "Total number of analysis result messages handled by status.",
		},
		[]string{"status"},
	)

	AnalysisResultHandleDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autoinspect_analysis_result_handle_duration_seconds",
			Help:    "Duration of analysis result handling in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"status"},
	)

	KafkaMessagesProducedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_kafka_messages_produced_total",
			Help: "Total number of Kafka messages produced.",
		},
		[]string{"topic", "status"},
	)

	KafkaMessagesConsumedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_kafka_messages_consumed_total",
			Help: "Total number of Kafka messages consumed by worker.",
		},
		[]string{"topic", "status"},
	)

	KafkaMessageProcessingDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "autoinspect_kafka_message_processing_duration_seconds",
			Help:    "Kafka message processing duration in seconds.",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"topic", "status"},
	)

	RepairRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_repair_requests_total",
			Help: "Total number of repair request actions.",
		},
		[]string{"action"},
	)

	ModelTrainingRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_model_training_requests_total",
			Help: "Total number of model training request actions.",
		},
		[]string{"action"},
	)

	CarServiceApplicationsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_car_service_applications_total",
			Help: "Total number of car service application actions.",
		},
		[]string{"action"},
	)

	ModelUploadsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_model_uploads_total",
			Help: "Total number of ML model artifact upload attempts.",
		},
		[]string{"status"},
	)

	ModelArtifactUploadBytesTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "autoinspect_model_artifact_upload_bytes_total",
			Help: "Total bytes uploaded for model artifacts.",
		},
		[]string{"artifact"},
	)

	ReadinessDependencyStatus = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "autoinspect_readiness_dependency_status",
			Help: "Readiness status of backend dependencies. 1 means ready, 0 means not ready.",
		},
		[]string{"service", "dependency"},
	)
)

func init() {
	prometheus.MustRegister(
		HTTPRequestsTotal,
		HTTPRequestDuration,
		HTTPRequestsInFlight,
		AnalysisJobsSubmittedTotal,
		AnalysisImagesUploadedTotal,
		AnalysisResultsReceivedTotal,
		AnalysisResultsHandledTotal,
		AnalysisResultHandleDuration,
		KafkaMessagesProducedTotal,
		KafkaMessagesConsumedTotal,
		KafkaMessageProcessingDuration,
		RepairRequestsTotal,
		ModelTrainingRequestsTotal,
		CarServiceApplicationsTotal,
		ModelUploadsTotal,
		ModelArtifactUploadBytesTotal,
		ReadinessDependencyStatus,
	)
}

func ObserveHTTPRequest(method, path string, status int, startedAt time.Time) {
	statusCode := strconv.Itoa(status)
	HTTPRequestsTotal.WithLabelValues(method, path, statusCode).Inc()
	HTTPRequestDuration.WithLabelValues(method, path, statusCode).Observe(time.Since(startedAt).Seconds())
}

func ObserveKafkaConsumed(topic string, err error, startedAt time.Time) {
	status := statusSuccess
	if err != nil {
		status = statusFailed
	}
	KafkaMessagesConsumedTotal.WithLabelValues(topic, status).Inc()
	KafkaMessageProcessingDuration.WithLabelValues(topic, status).Observe(time.Since(startedAt).Seconds())
}

func ObserveKafkaProduced(topic string, err error) {
	status := statusSuccess
	if err != nil {
		status = statusFailed
	}
	KafkaMessagesProducedTotal.WithLabelValues(topic, status).Inc()
}

func ObserveModelUpload(err error) {
	status := statusSuccess
	if err != nil {
		status = statusFailed
	}
	ModelUploadsTotal.WithLabelValues(status).Inc()
}

const (
	statusSuccess = "success"
	statusFailed  = "failed"
)
