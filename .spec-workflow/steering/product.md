---
inclusion: always
---

# Product Overview

## Vision
Provide a robust authentication proxy that integrates seamlessly into microservice architectures, enhancing both security and observability.

## Problem Statement
Address the lack of scalable, ready-to-integrate solutions for securing API endpoints and monitoring distributed applications seamlessly.

## Target Users
| Persona          | Role                | Needs                                  | Pain Points                          |
|------------------|---------------------|---------------------------------------|--------------------------------------|
| Application Dev  | Backend Developer  | Secure their API endpoints quickly    | Lack of proper authentication tooling |
| DevOps Engineer  | Infrastructure Ops | Monitor telemetry across services     | Poor observability into dependencies |

## Key Features
1. **Authentication Proxy** — OAuth2 and OpenTelemetry integration.
2. **Flexible Configuration** — Easily adjust rules and policies using Viper.
3. **Performance Optimized Telemetry** — High performance under load.

## Business Objectives
- [ ] Achieve deployment integration within 95% of backend environments.
- [ ] Ensure 99.9% uptime in monitored production setups.
- [ ] Simplify compliance efforts across all supported platforms.

## Success Metrics
| Metric           | Target   | Current   |
|------------------|----------|-----------|
| Uptime           | 99.9%    | TBD       |
| Response Time    | <200ms   | TBD       |
| Deployment Time  | <2 hours | TBD       |

## Domain Context
- **Authentication Proxy**: Acts as a middleware to add auth functionality to APIs.
- **Telemetry**: Enables in-depth application performance monitoring.

## Constraints
- **Regulatory**: Adhere to OAuth2/OpenID Connect.
- **Business**: Support multi-cloud setups.
- **Technical**: Must handle 10K RPS using minimal hardware.

## Out of Scope
- Frontend integrations or UI development.
- Database as a direct dependency.