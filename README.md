# OpenTelemetry Metrics Processor

A custom OpenTelemetry processor for statistical analysis and anomaly detection on metrics data.

## Overview

This processor analyzes metrics data in real-time, calculating statistical properties and detecting anomalies based on Z-score analysis. When a metric value deviates significantly from its historical mean, the processor flags it as an anomaly.

## Features

- **Statistical Analysis**: Maintains a sliding window of values for each metric and calculates statistical properties like mean and standard deviation.
- **Anomaly Detection**: Uses Z-score analysis to detect anomalies in metric values.
- **Configurable**: Customize the window size, anomaly threshold, and minimum data points required for analysis.
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
```

### Configuration Parameters

| Parameter | Description | Default |
|-----------|-------------|---------|
| `window_size` | Number of data points to consider for statistical calculations | 100 |
| `anomaly_threshold` | Number of standard deviations from the mean to consider a value anomalous | 3.0 |
| `min_data_points` | Minimum number of data points required before performing anomaly detection | 10 |

## Building

This processor is designed to be built into a custom OpenTelemetry Collector using the [OpenTelemetry Collector Builder](https://github.com/open-telemetry/opentelemetry-collector/tree/main/cmd/builder).

1. Ensure you have the OpenTelemetry Collector Builder installed:
   ```bash
   go install go.opentelemetry.io/collector/cmd/builder@latest
   ```

2. Build the custom collector using the provided `collector-builder.yaml`:
   ```bash
   builder --config=collector-builder.yaml
   ```

3. The built collector will be available in the `./bin` directory.

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

   exporters:
     debug:
       verbosity: detailed

   service:
     pipelines:
       metrics:
         receivers: [otlp]
         processors: [metricsprocessor]
         exporters: [debug]
   ```

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
6. Anomalies are logged to the console and can be extended to create new metrics or send alerts.

## Metric Type Handling

- **Gauge Metrics**: Direct value analysis
- **Sum Metrics**: Value analysis (could be extended to rate-of-change analysis)
- **Histogram Metrics**: Analysis of the average value
- **Summary Metrics**: Analysis of the average value

## License

[MIT License](LICENSE)
