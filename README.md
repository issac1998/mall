# High-Concurrency Seckill System

> A distributed high-concurrency seckill system based on Go language, designed as a personal project for interview demonstration, covering key capabilities such as high concurrency and distributed consistency.

## Project Overview

This project is a complete high-concurrency seckill system using microservice architecture, supporting seckill request processing at 10K+ QPS. The system features high availability, high concurrency, and data consistency, making it suitable for demonstrating distributed system design and development capabilities in interviews.

### Core Features

- ✅ **High Concurrency Processing**: Supports 10K+ QPS seckill requests while ensuring system stability
- ✅ **Data Consistency**: Ensures accurate product inventory, preventing overselling and underselling
- ✅ **System Reliability**: Fault-tolerant capabilities with fast failure recovery
- ✅ **Elastic Scaling**: Supports horizontal scaling to handle traffic surges
- ✅ **Complete Monitoring**: Integrated Prometheus + Grafana + Jaeger monitoring system

### Technology Stack

| Technology Domain | Selection | Version | Description |
|---------|------|------|------|
| Programming Language | Go | 1.21+ | High performance, concurrency-friendly, rich ecosystem |
| Web Framework | Gin | v1.9+ | Lightweight, high-performance HTTP framework |
| Database | MySQL | 8.0+ | Mature and stable relational database |
| Cache | Redis | 7.0+ | High-performance in-memory database |
| Message Queue | go-queue | latest | Lightweight Go message queue |
| Service Discovery | etcd | v3.5+ | Distributed key-value store for service registration and discovery |
| Containerization | Docker | 24.0+ | Application containerization deployment |

## Quick Start

### Environment Requirements

- Go 1.21+
- Docker & Docker Compose
- MySQL 8.0+
- Redis 7.0+

### Install Dependencies

```bash
# Clone the project
git clone https://github.com/issac1998/seckill-system.git
cd seckill-system

# Install Go dependencies
make deps

# Install development tools
make install-tools
```

### Start Services

```bash
# Start all dependency services (MySQL, Redis, etcd, etc.)
make up

# Initialize database
make db-init

# Start API service
make run
```

### Verify Services

```bash
# Health check
make health

# Or access directly
curl http://localhost:8080/health
```

## Project Structure

```
seckill-system/
├── cmd/                    # Application entry points
│   └── api/               # API service
├── internal/              # Internal private packages
│   ├── config/           # Configuration management
│   ├── model/            # Data models
│   ├── repository/       # Data access layer
│   ├── service/          # Business logic layer
│   ├── handler/          # HTTP handlers
│   ├── middleware/       # Middleware
│   └── utils/            # Utility functions
├── pkg/                   # Public packages
│   ├── logger/           # Logging component
│   ├── database/         # Database client
│   ├── cache/            # Cache component
│   └── errors/           # Error handling
├── configs/               # Configuration files
├── scripts/               # Script files
├── deployments/           # Deployment files
└── tests/                 # Test files
```

## Development Guide

### Code Standards

This project strictly follows Go language official code standards and best practices:

- Use `gofmt` for code formatting
- Follow Go naming conventions
- Complete error handling
- Structured logging
- Comprehensive unit testing

### Common Commands

```bash
# Code formatting
make fmt

# Code linting
make lint

# Run tests
make test

# Generate coverage report
make test-coverage

# Build application
make build

# Development mode (hot reload)
make dev
```

### Database Operations

```bash
# Initialize database
make db-init

# Run migrations
make db-migrate

# Rollback migrations
make db-rollback
```

### Docker Operations

```bash
# Start all services
make up

# Stop all services
make down

# View logs
make logs

# Build Docker images
make docker-build
```

## API Documentation

### Health Check

```http
GET /health
```

Response:
```json
{
  "code": 200,
  "message": "ok",
  "data": {
    "status": "healthy",
    "timestamp": "2024-01-01T12:00:00Z"
  }
}
```

### User Authentication

```http
POST /api/v1/auth/login
Content-Type: application/json

{
  "username": "test",
  "password": "123456"
}
```

### Seckill Interface

```http
POST /api/v1/seckill
Authorization: Bearer <token>
Content-Type: application/json

{
  "activity_id": 1,
  "product_id": 1,
  "quantity": 1
}
```

## Performance Testing

### Load Testing

```bash
# Use K6 for load testing
make load-test

# Benchmark testing
make bench
```

### Monitoring Metrics

- **QPS**: Queries per second
- **Response Time**: P50, P95, P99 latency
- **Error Rate**: 4xx, 5xx error ratio
- **System Resources**: CPU, memory, network utilization

## Deployment Guide

### Local Development Environment

```bash
# Start development environment
make up
make run
```

### Production Environment Deployment

```bash
# Build production image
make docker-build

# Deploy to production environment
docker-compose -f docker-compose.prod.yml up -d
```

## Monitoring and Operations

### Monitoring Dashboards

- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)
- **Jaeger**: http://localhost:16686

### Log Viewing

```bash
# View application logs
make logs

# View specific service logs
docker-compose logs -f api
```

## Troubleshooting

### Common Issues

1. **Service Startup Failure**
   - Check port usage: `lsof -i :8080`
   - Check configuration file: `configs/config.yaml`

2. **Database Connection Failure**
   - Check MySQL service status
   - Verify connection parameters

3. **Redis Connection Failure**
   - Check Redis service status
   - Verify connection configuration

### Performance Tuning

1. **Database Optimization**
   - Add appropriate indexes
   - Optimize query statements
   - Configure connection pool

2. **Cache Optimization**
   - Set reasonable TTL
   - Use cache preheating
   - Avoid cache avalanche

3. **Application Optimization**
   - Use connection pools
   - Asynchronous processing
   - Reasonable timeout settings

## Contributing

1. Fork the project
2. Create feature branch: `git checkout -b feature/new-feature`
3. Commit changes: `git commit -am 'Add new feature'`
4. Push branch: `git push origin feature/new-feature`
5. Submit Pull Request

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Contact

- Author: Isaac
- Email: isaac@example.com
- GitHub: https://github.com/issac1998/seckill-system

---

**Note**: This project is for learning and interview demonstration purposes only. It is not recommended for direct use in production environments.