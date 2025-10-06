package monitor

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracerConfig 链路追踪配置
type TracerConfig struct {
	ServiceName     string
	ServiceVersion  string
	Environment     string
	JaegerEndpoint  string
	SamplingRate    float64
	Enabled         bool
}

// Tracer 链路追踪器
type Tracer struct {
	config   *TracerConfig
	provider *trace.TracerProvider
	tracer   oteltrace.Tracer
}

// NewTracer 创建新的链路追踪器
func NewTracer(config *TracerConfig) (*Tracer, error) {
	if !config.Enabled {
		return &Tracer{
			config: config,
			tracer: otel.Tracer(config.ServiceName),
		}, nil
	}

	// 创建Jaeger导出器
	exporter, err := jaeger.New(
		jaeger.WithCollectorEndpoint(
			jaeger.WithEndpoint(config.JaegerEndpoint),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create jaeger exporter: %w", err)
	}

	// 创建资源
	res, err := resource.New(
		context.Background(),
		resource.WithAttributes(
			semconv.ServiceNameKey.String(config.ServiceName),
			semconv.ServiceVersionKey.String(config.ServiceVersion),
			semconv.DeploymentEnvironmentKey.String(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// 创建追踪提供者
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.SamplingRate)),
	)

	// 设置全局追踪提供者
	otel.SetTracerProvider(provider)
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	return &Tracer{
		config:   config,
		provider: provider,
		tracer:   tracer,
	}, nil
}

// StartSpan 开始一个新的span
func (t *Tracer) StartSpan(ctx context.Context, operationName string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}
	return t.tracer.Start(ctx, operationName, opts...)
}

// StartHTTPSpan 开始一个HTTP请求的span
func (t *Tracer) StartHTTPSpan(ctx context.Context, method, path string, r *http.Request) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	// 从HTTP头中提取追踪上下文
	ctx = otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(r.Header))

	// 开始span
	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("%s %s", method, path),
		oteltrace.WithSpanKind(oteltrace.SpanKindServer),
		oteltrace.WithAttributes(
			semconv.HTTPMethodKey.String(method),
			semconv.HTTPURLKey.String(r.URL.String()),
			semconv.HTTPSchemeKey.String(r.URL.Scheme),
			semconv.HTTPHostKey.String(r.Host),
			semconv.HTTPTargetKey.String(path),
			semconv.HTTPUserAgentKey.String(r.UserAgent()),
			semconv.HTTPClientIPKey.String(getClientIP(r)),
		),
	)

	return ctx, span
}

// StartDBSpan 开始一个数据库操作的span
func (t *Tracer) StartDBSpan(ctx context.Context, operation, table string) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("db.%s.%s", operation, table),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			semconv.DBSystemKey.String("mysql"),
			semconv.DBOperationKey.String(operation),
			semconv.DBSQLTableKey.String(table),
		),
	)

	return ctx, span
}

// StartRedisSpan 开始一个Redis操作的span
func (t *Tracer) StartRedisSpan(ctx context.Context, command string) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("redis.%s", command),
		oteltrace.WithSpanKind(oteltrace.SpanKindClient),
		oteltrace.WithAttributes(
			semconv.DBSystemKey.String("redis"),
			semconv.DBOperationKey.String(command),
		),
	)

	return ctx, span
}

// StartQueueSpan 开始一个队列操作的span
func (t *Tracer) StartQueueSpan(ctx context.Context, operation, queue string) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, fmt.Sprintf("queue.%s.%s", operation, queue),
		oteltrace.WithSpanKind(oteltrace.SpanKindProducer),
		oteltrace.WithAttributes(
			attribute.String("messaging.system", "go-queue"),
			attribute.String("messaging.operation", operation),
			attribute.String("messaging.destination", queue),
		),
	)

	return ctx, span
}

// StartSeckillSpan 开始一个秒杀操作的span
func (t *Tracer) StartSeckillSpan(ctx context.Context, activityID uint64, userID uint64) (context.Context, oteltrace.Span) {
	if !t.config.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	ctx, span := t.tracer.Start(ctx, "seckill.execute",
		oteltrace.WithAttributes(
			attribute.Int64("seckill.activity_id", int64(activityID)),
			attribute.Int64("seckill.user_id", int64(userID)),
		),
	)

	return ctx, span
}

// AddSpanAttributes 添加span属性
func (t *Tracer) AddSpanAttributes(span oteltrace.Span, attrs ...attribute.KeyValue) {
	if !t.config.Enabled {
		return
	}
	span.SetAttributes(attrs...)
}

// AddSpanEvent 添加span事件
func (t *Tracer) AddSpanEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	if !t.config.Enabled {
		return
	}
	span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

// RecordError 记录错误
func (t *Tracer) RecordError(span oteltrace.Span, err error) {
	if !t.config.Enabled || err == nil {
		return
	}
	span.RecordError(err)
	span.SetStatus(oteltrace.StatusError, err.Error())
}

// SetSpanStatus 设置span状态
func (t *Tracer) SetSpanStatus(span oteltrace.Span, code oteltrace.StatusCode, description string) {
	if !t.config.Enabled {
		return
	}
	span.SetStatus(code, description)
}

// InjectHTTPHeaders 将追踪上下文注入HTTP头
func (t *Tracer) InjectHTTPHeaders(ctx context.Context, headers http.Header) {
	if !t.config.Enabled {
		return
	}
	otel.GetTextMapPropagator().Inject(ctx, propagation.HeaderCarrier(headers))
}

// ExtractHTTPHeaders 从HTTP头中提取追踪上下文
func (t *Tracer) ExtractHTTPHeaders(ctx context.Context, headers http.Header) context.Context {
	if !t.config.Enabled {
		return ctx
	}
	return otel.GetTextMapPropagator().Extract(ctx, propagation.HeaderCarrier(headers))
}

// Shutdown 关闭追踪器
func (t *Tracer) Shutdown(ctx context.Context) error {
	if !t.config.Enabled || t.provider == nil {
		return nil
	}
	return t.provider.Shutdown(ctx)
}

// SpanFromContext 从上下文中获取span
func (t *Tracer) SpanFromContext(ctx context.Context) oteltrace.Span {
	return oteltrace.SpanFromContext(ctx)
}

// ContextWithSpan 将span添加到上下文中
func (t *Tracer) ContextWithSpan(ctx context.Context, span oteltrace.Span) context.Context {
	return oteltrace.ContextWithSpan(ctx, span)
}

// TraceID 获取追踪ID
func (t *Tracer) TraceID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().TraceID().String()
	}
	return ""
}

// SpanID 获取SpanID
func (t *Tracer) SpanID(ctx context.Context) string {
	span := oteltrace.SpanFromContext(ctx)
	if span.SpanContext().IsValid() {
		return span.SpanContext().SpanID().String()
	}
	return ""
}

// WithTimeout 创建带超时的上下文和span
func (t *Tracer) WithTimeout(ctx context.Context, operationName string, timeout time.Duration) (context.Context, oteltrace.Span, context.CancelFunc) {
	ctx, span := t.StartSpan(ctx, operationName)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	
	// 添加超时属性
	t.AddSpanAttributes(span, attribute.String("timeout", timeout.String()))
	
	return ctx, span, cancel
}

// Middleware 创建HTTP中间件
func (t *Tracer) Middleware() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ctx, span := t.StartHTTPSpan(r.Context(), r.Method, r.URL.Path, r)
			defer span.End()

			// 创建响应写入器包装器
			rw := &responseWriter{
				ResponseWriter: w,
				statusCode:     http.StatusOK,
			}

			// 处理请求
			r = r.WithContext(ctx)
			next.ServeHTTP(rw, r)

			// 记录响应信息
			t.AddSpanAttributes(span,
				semconv.HTTPStatusCodeKey.Int(rw.statusCode),
				semconv.HTTPResponseSizeKey.Int64(rw.bytesWritten),
			)

			// 设置状态
			if rw.statusCode >= 400 {
				t.SetSpanStatus(span, oteltrace.StatusError, fmt.Sprintf("HTTP %d", rw.statusCode))
			} else {
				t.SetSpanStatus(span, oteltrace.StatusOK, "")
			}
		})
	}
}

// responseWriter 响应写入器包装器
type responseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	n, err := rw.ResponseWriter.Write(b)
	rw.bytesWritten += int64(n)
	return n, err
}

// getClientIP 获取客户端IP
func getClientIP(r *http.Request) string {
	// 检查X-Forwarded-For头
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		return xff
	}
	
	// 检查X-Real-IP头
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}
	
	// 使用RemoteAddr
	return r.RemoteAddr
}

// DefaultTracerConfig 默认追踪器配置
func DefaultTracerConfig() *TracerConfig {
	return &TracerConfig{
		ServiceName:     "seckill-system",
		ServiceVersion:  "1.0.0",
		Environment:     "development",
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		SamplingRate:    1.0,
		Enabled:         true,
	}
}