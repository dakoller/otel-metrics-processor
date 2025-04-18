# OpenTelemetry Metrics Processor

A custom OpenTelemetry processor for statistical analysis and anomaly detection on metrics data.

## Overview

This processor analyzes metrics data in real-time, calculating statistical properties and detecting anomalies based on Z-score analysis. When a metric value deviates significantly from its historical mean, the processor flags it as an anomaly.

## Features

- **Statistical Analysis**: Maintains a sliding window of values for each metric and calculates statistical properties like mean and standard deviation.
- **Anomaly Detection**: Uses Z-score analysis to detect anomalies in metric values.
- **Anomaly Logging**: Creates detailed log entries for detected anomalies with configurable severity levels.
- **Configurable**: Customize the window size, anomaly threshold, minimum data points required for analysis, and logging behavior.
- **Support for All Metric Types**: Handles gauge, sum, histogram, and summary metrics.
- **Thread-Safe**: Designed for concurrent processing of metrics data.

## Configuration

The processor supports the following configuration options:

```yaml
processors:
  metricsprocessor:
    # Number of data points to consider for statistical calculations
    window_size: 100
    # Number of standard deviations from the mean to consider a value anomalous
    anomaly_threshold: 3.0
    # Minimum number of data points required before performing anomaly detection
    min_data_points: 10
    # Whether to create log entries for anomalies
    log_anomalies: true
    # Severity level for anomaly log entries (INFO, WARN, ERROR)
    log_severity: WARN
```

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `window_size` | Number of data points to consider for statistical calculations | 100 |
| `anomaly_threshold` | Number of standard deviations from the mean to consider a value anomalous | 3.0 |
| `min_data_points` | Minimum number of data points required before performing anomaly detection | 10 |
| `log_anomalies` | Whether to create log entries for anomalies | true |
| `log_severity` | Severity level for anomaly log entries (INFO, WARN, ERROR) | WARN |

## Building

This processor is designed to be built into a custom OpenTelemetry Collector using the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).

1. Ensure you have the OpenTelemetry Collector Builder installed:
   ```bash
   go install go.opentelemetry.io/collector/cmd/builder@latest
   ```

2. Configure the collector builder in `collector-builder.yaml`:
   ```yaml
   dist:
     name: otelcol-custom
     output_path: ./bin
   exporters:
     - gomod: go.opentelemetry.io/collector/exporter/debugexporter v0.96.0
   processors:
     - gomod: github.com/yourusername/otel-metrics-processor v0.0.1
   receivers:
     - gomod: go.opentelemetry.io/collector/receiver/otlpreceiver v0.96.0
   ```

3. Build the custom collector:
   ```bash
   builder --config=collector-builder.yaml
   ```

4. The built collector will be available in the `./bin` directory.

## Usage

1. Configure the processor in your OpenTelemetry Collector configuration file:
   ```yaml
   receivers:
     otlp:
       protocols:
         grpc:
         http:

   processors:
     metricsprocessor:
       window_size: 100
       anomaly_threshold: 3.0
       min_data_points: 10
       log_anomalies: true
       log_severity: WARN

   exporters:
     debug:
       verbosity: detailed

   service:
     pipelines:
       metrics:
         receivers: [otlp]
         processors: [metricsprocessor]
         exporters: [debug]
       logs:
         receivers: [otlp]
         exporters: [debug]
   ```

   Note: The logs pipeline is required to receive the anomaly logs generated by the metrics processor.

2. Run the custom collector:
   ```bash
   ./bin/otelcol-custom --config=otel-config.yaml
   ```

## How It Works

1. The processor receives metrics from the configured receivers.
2. For each metric, it maintains a sliding window of recent values.
3. It calculates the mean and standard deviation of the values in the window.
4. When a new value arrives, it calculates the Z-score: `|value - mean| / stdDev`.
5. If the Z-score exceeds the configured threshold, the value is flagged as an anomaly.
6. When an anomaly is detected:
   - It is logged to the console
   - If `log_anomalies` is enabled, a log entry is created with the configured severity level
   - The log entry includes detailed information about the anomaly, including the metric name, value, mean, standard deviation, and Z-score
   - The log entry is sent to the logs pipeline of the OpenTelemetry Collector

## Metric Type Handling

- **Gauge Metrics**: Direct value analysis
- **Sum Metrics**: Value analysis (could be extended to rate-of-change analysis)
- **Histogram Metrics**: Analysis of the average value
- **Summary Metrics**: Analysis of the average value

## License

[MIT License](LICENSE)
