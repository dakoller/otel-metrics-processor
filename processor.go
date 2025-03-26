package processor

import (
	"context"
	"fmt"
	"math"
	"sync"
	"time"

	"go.opentelemetry.io/collector/pdata/pcommon"
	"go.opentelemetry.io/collector/pdata/plog"
	"go.opentelemetry.io/collector/pdata/pmetric"
	"go.opentelemetry.io/collector/consumer"
)

// LogSeverity defines the severity level for anomaly log entries
type LogSeverity string

const (
	// LogSeverityInfo represents INFO severity level
	LogSeverityInfo LogSeverity = "INFO"
	// LogSeverityWarn represents WARN severity level
	LogSeverityWarn LogSeverity = "WARN"
	// LogSeverityError represents ERROR severity level
	LogSeverityError LogSeverity = "ERROR"
)

// Config defines the configuration for the metrics processor
type Config struct {
	// WindowSize is the number of data points to consider for statistical calculations
	WindowSize int `mapstructure:"window_size"`
	// AnomalyThreshold is the number of standard deviations from the mean to consider a value anomalous
	AnomalyThreshold float64 `mapstructure:"anomaly_threshold"`
	// MinDataPoints is the minimum number of data points required before performing anomaly detection
	MinDataPoints int `mapstructure:"min_data_points"`
	// LogAnomalies determines whether to create log entries for anomalies
	LogAnomalies bool `mapstructure:"log_anomalies"`
	// LogSeverity is the severity level for anomaly log entries
	LogSeverity LogSeverity `mapstructure:"log_severity"`
}

// metricStats holds statistical data for a specific metric
type metricStats struct {
	values       []float64
	sum          float64
	sumOfSquares float64
	count        int
	mean         float64
	stdDev       float64
	lastUpdate   time.Time
	mu           sync.RWMutex
}

// metricsProcessor implements the metrics processor for statistical analysis and anomaly detection
type metricsProcessor struct {
	nextConsumer consumer.Metrics
	logExporter  consumer.Logs
	config       *Config
	// metricsData maps metric names to their statistical data
	metricsData map[string]*metricStats
	mu          sync.RWMutex
}

func newMetricsProcessor(next consumer.Metrics, logExporter consumer.Logs, config *Config) *metricsProcessor {
	if config == nil {
		config = &Config{
			WindowSize:       100,
			AnomalyThreshold: 3.0,
			MinDataPoints:    10,
			LogAnomalies:     true,
			LogSeverity:      LogSeverityWarn,
		}
	}
	
	return &metricsProcessor{
		nextConsumer: next,
		logExporter:  logExporter,
		config:       config,
		metricsData:  make(map[string]*metricStats),
	}
}

func (mp *metricsProcessor) Capabilities() consumer.Capabilities {
	return consumer.Capabilities{MutatesData: true}
}

func (mp *metricsProcessor) ConsumeMetrics(ctx context.Context, metrics pmetric.Metrics) error {
	// Iterate over metrics and compute statistical features
	rms := metrics.ResourceMetrics()
	for i := 0; i < rms.Len(); i++ {
		rm := rms.At(i)
		scopeMetrics := rm.ScopeMetrics()
		
		for j := 0; j < scopeMetrics.Len(); j++ {
			sm := scopeMetrics.At(j)
			ms := sm.Metrics()
			
			for k := 0; k < ms.Len(); k++ {
				m := ms.At(k)
				metricName := m.Name()
				
				switch m.Type() {
				case pmetric.MetricTypeGauge:
					mp.handleGaugeMetric(metricName, m)
				case pmetric.MetricTypeSum:
					mp.handleSumMetric(metricName, m)
				case pmetric.MetricTypeHistogram:
					mp.handleHistogramMetric(metricName, m)
				case pmetric.MetricTypeSummary:
					mp.handleSummaryMetric(metricName, m)
				}
			}
		}
	}

	// Pass metrics to next consumer after processing
	return mp.nextConsumer.ConsumeMetrics(ctx, metrics)
}

func (mp *metricsProcessor) handleGaugeMetric(name string, metric pmetric.Metric) {
	dp := metric.Gauge().DataPoints()
	for l := 0; l < dp.Len(); l++ {
		point := dp.At(l)
		value := point.DoubleValue()
		timestamp := point.Timestamp().AsTime()
		
		isAnomaly := mp.updateStats(name, value, timestamp)
		if isAnomaly {
			fmt.Printf("ANOMALY DETECTED: Metric %s with value %f is outside normal range\n", name, value)
			
			// Create log entry for the anomaly if configured
			if mp.config.LogAnomalies && mp.logExporter != nil {
				mp.createAnomalyLogEntry(name, value, timestamp)
			}
		}
	}
}

func (mp *metricsProcessor) handleSumMetric(name string, metric pmetric.Metric) {
	dp := metric.Sum().DataPoints()
	for l := 0; l < dp.Len(); l++ {
		point := dp.At(l)
		value := point.DoubleValue()
		timestamp := point.Timestamp().AsTime()
		
		// For cumulative sums, we might want to calculate the rate of change
		// rather than analyzing the raw value
		isAnomaly := mp.updateStats(name, value, timestamp)
		if isAnomaly {
			fmt.Printf("ANOMALY DETECTED: Metric %s with value %f is outside normal range\n", name, value)
			
			// Create log entry for the anomaly if configured
			if mp.config.LogAnomalies && mp.logExporter != nil {
				mp.createAnomalyLogEntry(name, value, timestamp)
			}
		}
	}
}

func (mp *metricsProcessor) handleHistogramMetric(name string, metric pmetric.Metric) {
	dp := metric.Histogram().DataPoints()
	for l := 0; l < dp.Len(); l++ {
		point := dp.At(l)
		count := point.Count()
		sum := point.Sum()
		timestamp := point.Timestamp().AsTime()
		
		if count > 0 {
			// Use the average value for anomaly detection
			avg := sum / float64(count)
			isAnomaly := mp.updateStats(name, avg, timestamp)
			if isAnomaly {
				fmt.Printf("ANOMALY DETECTED: Histogram metric %s with average %f is outside normal range\n", name, avg)
				
				// Create log entry for the anomaly if configured
				if mp.config.LogAnomalies && mp.logExporter != nil {
					mp.createAnomalyLogEntry(name, avg, timestamp)
				}
			}
		}
	}
}

func (mp *metricsProcessor) handleSummaryMetric(name string, metric pmetric.Metric) {
	dp := metric.Summary().DataPoints()
	for l := 0; l < dp.Len(); l++ {
		point := dp.At(l)
		count := point.Count()
		sum := point.Sum()
		timestamp := point.Timestamp().AsTime()
		
		if count > 0 {
			// Use the average value for anomaly detection
			avg := sum / float64(count)
			isAnomaly := mp.updateStats(name, avg, timestamp)
			if isAnomaly {
				fmt.Printf("ANOMALY DETECTED: Summary metric %s with average %f is outside normal range\n", name, avg)
				
				// Create log entry for the anomaly if configured
				if mp.config.LogAnomalies && mp.logExporter != nil {
					mp.createAnomalyLogEntry(name, avg, timestamp)
				}
			}
		}
	}
}

// updateStats updates the statistical data for a metric and checks for anomalies
// Returns true if the value is anomalous, false otherwise
func (mp *metricsProcessor) updateStats(metricName string, value float64, timestamp time.Time) bool {
	mp.mu.Lock()
	stats, exists := mp.metricsData[metricName]
	if !exists {
		stats = &metricStats{
			values:     make([]float64, 0, mp.config.WindowSize),
			lastUpdate: timestamp,
		}
		mp.metricsData[metricName] = stats
	}
	mp.mu.Unlock()
	
	stats.mu.Lock()
	defer stats.mu.Unlock()
	
	// Check if this is an anomaly before adding it to our dataset
	isAnomaly := false
	if stats.count >= mp.config.MinDataPoints {
		zScore := math.Abs(value - stats.mean) / stats.stdDev
		if !math.IsNaN(zScore) && zScore > mp.config.AnomalyThreshold {
			isAnomaly = true
		}
	}
	
	// Add the new value to our dataset
	if len(stats.values) >= mp.config.WindowSize {
		// Remove the oldest value from our calculations
		oldestValue := stats.values[0]
		stats.sum -= oldestValue
		stats.sumOfSquares -= oldestValue * oldestValue
		stats.count--
		
		// Shift values to remove the oldest
		stats.values = append(stats.values[1:], value)
	} else {
		stats.values = append(stats.values, value)
	}
	
	// Update statistics
	stats.sum += value
	stats.sumOfSquares += value * value
	stats.count++
	stats.lastUpdate = timestamp
	
	// Recalculate mean and standard deviation
	stats.mean = stats.sum / float64(stats.count)
	
	// Calculate variance and standard deviation
	if stats.count > 1 {
		variance := (stats.sumOfSquares - stats.sum*stats.sum/float64(stats.count)) / float64(stats.count-1)
		stats.stdDev = math.Sqrt(math.Max(0, variance)) // Ensure non-negative
	} else {
		stats.stdDev = 0
	}
	
	return isAnomaly
}

// GetMetricStats returns the current statistical data for a specific metric
func (mp *metricsProcessor) GetMetricStats(metricName string) (mean, stdDev float64, count int, exists bool) {
	mp.mu.RLock()
	defer mp.mu.RUnlock()
	
	stats, exists := mp.metricsData[metricName]
	if !exists {
		return 0, 0, 0, false
	}
	
	stats.mu.RLock()
	defer stats.mu.RUnlock()
	
	return stats.mean, stats.stdDev, stats.count, true
}

// createAnomalyLogEntry creates and exports a log entry for an anomaly
func (mp *metricsProcessor) createAnomalyLogEntry(metricName string, value float64, timestamp time.Time) {
	// Create a new logs collection
	logs := plog.NewLogs()
	
	// Add a resource
	resource := logs.ResourceLogs().AppendEmpty()
	resource.Resource().Attributes().PutStr("service.name", "metrics-processor")
	
	// Add a scope
	scope := resource.ScopeLogs().AppendEmpty()
	
	// Add a log record
	record := scope.LogRecords().AppendEmpty()
	record.SetTimestamp(pcommon.NewTimestampFromTime(timestamp))
	
	// Set severity based on configuration
	switch mp.config.LogSeverity {
	case LogSeverityInfo:
		record.SetSeverityNumber(plog.SeverityNumberInfo)
		record.SetSeverityText("INFO")
	case LogSeverityWarn:
		record.SetSeverityNumber(plog.SeverityNumberWarn)
		record.SetSeverityText("WARN")
	case LogSeverityError:
		record.SetSeverityNumber(plog.SeverityNumberError)
		record.SetSeverityText("ERROR")
	default:
		record.SetSeverityNumber(plog.SeverityNumberWarn)
		record.SetSeverityText("WARN")
	}
	
	// Get the mean and standard deviation for context
	mean, stdDev, _, _ := mp.GetMetricStats(metricName)
	zScore := math.Abs(value - mean) / stdDev
	
	// Set the log message
	message := fmt.Sprintf("ANOMALY DETECTED: Metric %s with value %f is outside normal range (z-score: %.2f)", 
		metricName, value, zScore)
	record.Body().SetStr(message)
	
	// Add attributes
	attrs := record.Attributes()
	attrs.PutStr("metric.name", metricName)
	attrs.PutDouble("metric.value", value)
	attrs.PutDouble("metric.mean", mean)
	attrs.PutDouble("metric.stddev", stdDev)
	attrs.PutDouble("metric.zscore", zScore)
	attrs.PutStr("anomaly.type", "statistical")
	attrs.PutStr("processor.name", "metricsprocessor")
	
	// Export the log
	if err := mp.logExporter.ConsumeLogs(context.Background(), logs); err != nil {
		fmt.Printf("Failed to export anomaly log: %v\n", err)
	}
}
