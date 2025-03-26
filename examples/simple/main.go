package main

import (
	"context"
	"log"
	"math/rand"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetricgrpc"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"
)

func main() {
	// Initialize random seed
	rand.Seed(time.Now().UnixNano())

	// Create a resource with service information
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName("example-service"),
			semconv.ServiceVersion("1.0.0"),
		),
	)
	if err != nil {
		log.Fatalf("Failed to create resource: %v", err)
	}

	// Create OTLP exporter
	exporter, err := otlpmetricgrpc.New(context.Background(),
		otlpmetricgrpc.WithEndpoint("localhost:4317"),
		otlpmetricgrpc.WithInsecure(),
	)
	if err != nil {
		log.Fatalf("Failed to create exporter: %v", err)
	}

	// Create meter provider
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithResource(res),
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(exporter, sdkmetric.WithInterval(1*time.Second))),
	)
	defer func() {
		if err := meterProvider.Shutdown(context.Background()); err != nil {
			log.Fatalf("Failed to shutdown meter provider: %v", err)
		}
	}()

	// Set global meter provider
	otel.SetMeterProvider(meterProvider)

	// Get a meter
	meter := meterProvider.Meter("example-meter")

	// Create instruments
	counter, err := meter.Int64Counter("example.counter", metric.WithDescription("Example counter"))
	if err != nil {
		log.Fatalf("Failed to create counter: %v", err)
	}

	gauge, err := meter.Float64ObservableGauge("example.gauge", metric.WithDescription("Example gauge"))
	if err != nil {
		log.Fatalf("Failed to create gauge: %v", err)
	}

	// Register callback for observable gauge
	_, err = meter.RegisterCallback(
		func(ctx context.Context, observer metric.Observer) error {
			// Generate a random value between 0 and 100
			value := rand.Float64() * 100
			
			// Every 10th value, generate an anomaly (value between 300 and 400)
			if rand.Intn(10) == 0 {
				value = 300 + rand.Float64()*100
				log.Printf("Generating anomaly value: %f", value)
			} else {
				log.Printf("Generating normal value: %f", value)
			}
			
			observer.ObserveFloat64(gauge, value)
			return nil
		},
		gauge,
	)
	if err != nil {
		log.Fatalf("Failed to register callback: %v", err)
	}

	log.Println("Starting to send metrics. Press Ctrl+C to stop.")
	log.Println("The example will generate normal values between 0-100 and anomalies between 300-400 (every ~10th value).")

	// Increment counter every second
	for i := 0; i < 100; i++ {
		counter.Add(context.Background(), 1)
		time.Sleep(1 * time.Second)
	}
}
