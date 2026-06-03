package telemetry

import (
	"context"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

type Config struct {
	ServiceName      string
	Endpoint         string
	TelemetryEnabled bool
}

type Instrumentation struct {
	enabled        bool
	requestCounter metric.Int64Counter
	latency        metric.Float64Histogram
}

func Init(ctx context.Context, cfg Config) (*Instrumentation, func(context.Context) error, error) {
	if !cfg.TelemetryEnabled {
		return &Instrumentation{}, func(context.Context) error { return nil }, nil
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
		),
	)
	if err != nil {
		return nil, nil, err
	}

	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(cfg.Endpoint),
	)
	if err != nil {
		return nil, nil, err
	}

	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(cfg.Endpoint),
	)
	if err != nil {
		return nil, nil, err
	}

	tracerProvider := trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)
	meterProvider := sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter, sdkmetric.WithInterval(5*time.Second))),
		sdkmetric.WithResource(res),
	)

	otel.SetTracerProvider(tracerProvider)
	otel.SetMeterProvider(meterProvider)

	meter := otel.Meter("todo-api")
	requestCounter, err := meter.Int64Counter("http.server.request.count")
	if err != nil {
		return nil, nil, err
	}
	latency, err := meter.Float64Histogram("http.server.request.duration.ms")
	if err != nil {
		return nil, nil, err
	}

	shutdown := func(ctx context.Context) error {
		if err := meterProvider.Shutdown(ctx); err != nil {
			return err
		}
		return tracerProvider.Shutdown(ctx)
	}

	return &Instrumentation{
		enabled:        true,
		requestCounter: requestCounter,
		latency:        latency,
	}, shutdown, nil
}

func (i *Instrumentation) HTTPMiddleware(next http.Handler) http.Handler {
	if i == nil || !i.enabled {
		return next
	}

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ctx, span := otel.Tracer("todo-api").Start(r.Context(), r.Method+" "+r.URL.Path)
		defer span.End()

		start := time.Now()
		rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}
		next.ServeHTTP(rw, r.WithContext(ctx))

		attrs := []attribute.KeyValue{
			attribute.String("method", r.Method),
			attribute.String("path", r.URL.Path),
			attribute.Int("status_code", rw.statusCode),
		}
		i.requestCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
		i.latency.Record(ctx, float64(time.Since(start).Milliseconds()), metric.WithAttributes(attrs...))
	})
}

type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *responseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}
