---
inclusion: always
---

# Technology Stack

## Core Technologies

### Languages
| Language         | Version | Purpose                  |
|------------------|---------|--------------------------|
| Go               | 1.26    | Backend API logic        |
| JavaScript       | N/A     | Documentation generation |

### Frameworks & Libraries
| Name                        | Version  | Purpose                   | Alternatives Considered |
|-----------------------------|----------|---------------------------|--------------------------|
| Labstack Echo               | 4.15.x   | Web framework (Go)        | Gin, Fiber              |
| Prometheus Client Go        | 1.23.x   | Metrics collection        | OpenMetrics             |
| OpenTelemetry SDK           | 1.42.x   | Observability telemetry   | jaeger-client, Zipkin   |
| Docusaurus (JS)             | 3.7.x    | Documentation framework   | Hugo, Jekyll            |
| React                       | 18.3.x   | Interactive documentation | Vue.js, Svelte          |

### Infrastructure
| Service         | Provider   | Purpose                           |
|-----------------|------------|-----------------------------------|
| Hosting         | On-Prem    | Backend services                 |
| Observability   | OpenTelemetry | Distributed tracing, telemetry |

## Development Tools

### Build & Package Management
- **Package Manager**: Go Mod (Go), npm (JS)
- **Build Tool**: Go native tooling, docusaurus build (docs)
- **Version**: Using LTS/support expectations

### Code Quality
- **Linter**: golangci-lint — `.golangci.yaml`
- **Formatter**: `gofmt`
- **Type System**: Static typing in Go

### Testing
- **Test Framework**: `go test`
- **E2E Framework**: None defined yet.
- **Coverage Target**: 80% Minimum Baseline.

## Technical Constraints

### Performance Requirements
- **Latency**: <200ms per HTTP request
- **Throughput**: Up to 10K requests/second target scale.
- **Memory**: Minimal overhead observed <1GB in stress tests.

### Compatibility
- **Cloud**: Runs portable, even multi-cloud configurations AWS edge 
- **Node Docs**: unrelated configure bridges.