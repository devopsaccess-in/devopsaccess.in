---
name: otel-instrumentation
description: Use this skill whenever the user asks to add OpenTelemetry instrumentation, set up tracing, configure OTel exporters, or instrument a Go or Next.js service for observability. Covers both server-side (Go) and client-side (Next.js browser + server) patterns. Triggers on "OTel", "OpenTelemetry", "tracing", "spans", "otlp", "instrument this".
---

# OpenTelemetry Instrumentation in DevOps Access

Observability is the product. Every service is instrumented end-to-end.

## Go services — standard setup

### Initialization in main.go

```go
import (
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.26.0"
)

func initTracer(ctx context.Context, cfg Config) (func(context.Context) error, error) {
	exp, err := otlptracegrpc.New(ctx,
		otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
		otlptracegrpc.WithInsecure(), // use WithTLSCredentials in prod
	)
	if err != nil {
		return nil, fmt.Errorf("creating otlp exporter: %w", err)
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceName(cfg.ServiceName),
			semconv.ServiceVersion(cfg.Version),
			semconv.DeploymentEnvironment(cfg.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("creating resource: %w", err)
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exp),
		sdktrace.WithResource(res),
		sdktrace.WithSampler(sdktrace.ParentBased(
			sdktrace.TraceIDRatioBased(cfg.SampleRate), // 1.0 dev, 0.1 prod
		)),
	)
	otel.SetTracerProvider(tp)

	return tp.Shutdown, nil
}
```

### HTTP middleware

```go
import "go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp"

router := chi.NewRouter()
router.Use(otelhttp.NewMiddleware("api-server"))
```

### pgx instrumentation

```go
import "github.com/exaring/otelpgx"

cfg, _ := pgxpool.ParseConfig(dsn)
cfg.ConnConfig.Tracer = otelpgx.NewTracer()
```

### Manual spans for business logic

```go
var tracer = otel.Tracer("tenants")

func (s *Service) GetTenant(ctx context.Context, id string) (*Tenant, error) {
	ctx, span := tracer.Start(ctx, "tenants.Get",
		trace.WithAttributes(attribute.String("tenant.id", id)),
	)
	defer span.End()

	t, err := s.repo.Get(ctx, id)
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
		return nil, err
	}

	span.SetAttributes(attribute.String("tenant.tier", t.Tier))
	return t, nil
}
```

## Next.js — server-side instrumentation

In `apps/<web|dashboard>/instrumentation.ts` (Next.js auto-loads this file):

```typescript
export async function register() {
  if (process.env.NEXT_RUNTIME === 'nodejs') {
    const { NodeSDK } = await import('@opentelemetry/sdk-node');
    const { OTLPTraceExporter } = await import('@opentelemetry/exporter-trace-otlp-http');
    const { getNodeAutoInstrumentations } = await import('@opentelemetry/auto-instrumentations-node');

    const sdk = new NodeSDK({
      serviceName: 'devopsaccess-web',
      traceExporter: new OTLPTraceExporter({
        url: `${process.env.OTEL_EXPORTER_OTLP_ENDPOINT}/v1/traces`,
      }),
      instrumentations: [getNodeAutoInstrumentations()],
    });

    sdk.start();
  }
}
```

Next.js 14: add `experimental.instrumentationHook = true` to `next.config.js`.
Next.js 15+: instrumentation is auto-loaded, no flag needed.

## Hard rules

1. Service name attribute MUST match pattern `devopsaccess-<svc>` (`api`, `web`, `dashboard`, `ingestor`, etc.)
2. Environment attribute MUST be one of: `prod`, `staging`, `dev`, `local`.
3. Sample rate: 100% in dev/staging, 10% in prod.
4. Every outbound HTTP call MUST propagate context (use `otelhttp.DefaultClient` or wrap with `otelhttp.NewTransport`).
5. Every gRPC call MUST use `otelgrpc.UnaryClientInterceptor` / `otelgrpc.UnaryServerInterceptor`.
6. Never log sensitive data as span attributes (passwords, tokens, PII, card numbers).
7. Span names: `<module>.<Operation>`, e.g., `tenants.Get`, `payments.Charge`.

## Common pitfalls

- OTel SDK must be initialized BEFORE any instrumented package is imported. In Go, do it first in `main`. In Next.js, use `instrumentation.ts`.
- `context.Background()` breaks span parenting. Always thread request context through.
- Batched exporter drops spans on crash. For dev, use `sdktrace.WithSyncer` instead of `WithBatcher`.
- Resource attributes cost memory per span. Keep them to essentials.
- `otelhttp.NewMiddleware(name)` — `name` appears as the span name prefix. Use the service role ("api-server", "ingestor"), not a random string.
