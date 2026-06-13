# eks-cost-sentinel

> A Kubernetes operator that continuously tracks and annotates workloads with their estimated AWS compute cost.

[![CNCF Landscape](https://img.shields.io/badge/CNCF-landscape-blue)](https://landscape.cncf.io/)
[![License: Apache 2.0](https://img.shields.io/badge/License-Apache%202.0-blue.svg)](LICENSE)
[![Go Report Card](https://goreportcard.com/badge/github.com/goutham-annem/eks-cost-sentinel)](https://goreportcard.com/report/github.com/goutham-annem/eks-cost-sentinel)

## Overview

`eks-cost-sentinel` is a Kubernetes operator built with [controller-runtime](https://github.com/kubernetes-sigs/controller-runtime) that:

- Watches `Deployment`, `StatefulSet`, and `DaemonSet` resources
- Queries the AWS Pricing API to estimate hourly/monthly compute cost per workload
- Annotates resources with cost metadata (`cost-sentinel.io/hourly-cost`, `cost-sentinel.io/monthly-estimate`)
- Exposes a `WorkloadCostReport` CRD for aggregate cluster-wide cost visibility
- Emits Prometheus metrics for cost dashboards (Grafana-ready)

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Kubernetes Cluster                      │
│                                                           │
│  ┌─────────────────┐      ┌──────────────────────────┐  │
│  │  Deployment /   │      │   eks-cost-sentinel       │  │
│  │  StatefulSet /  │◄────►│   (operator pod)          │  │
│  │  DaemonSet      │      │                           │  │
│  └─────────────────┘      │  ┌──────────────────────┐│  │
│                            │  │ WorkloadCostReport   ││  │
│  ┌─────────────────┐      │  │ CRD                  ││  │
│  │  Node (EC2      │      │  └──────────────────────┘│  │
│  │  instance type) │◄────►│                           │  │
│  └─────────────────┘      └───────────┬──────────────┘  │
│                                        │                  │
└────────────────────────────────────────┼──────────────────┘
                                         │
                              ┌──────────▼──────────┐
                              │  AWS Pricing API     │
                              │  (ec2 spot/on-demand)│
                              └─────────────────────┘
```

## Quick Start

```bash
# Install CRDs
kubectl apply -f https://raw.githubusercontent.com/goutham-annem/eks-cost-sentinel/main/config/crd/

# Deploy the operator
kubectl apply -f https://raw.githubusercontent.com/goutham-annem/eks-cost-sentinel/main/config/manager/

# Check annotations on your deployments
kubectl get deployment -n default -o jsonpath='{.items[*].metadata.annotations}' | jq .
```

## Example Annotations

After the operator reconciles, your workloads get:

```yaml
metadata:
  annotations:
    cost-sentinel.io/instance-type: "m5.xlarge"
    cost-sentinel.io/hourly-cost: "0.192"
    cost-sentinel.io/monthly-estimate: "140.16"
    cost-sentinel.io/last-updated: "2026-06-13T10:00:00Z"
    cost-sentinel.io/pricing-model: "on-demand"
```

## WorkloadCostReport CRD

```yaml
apiVersion: cost-sentinel.io/v1alpha1
kind: WorkloadCostReport
metadata:
  name: cluster-cost-summary
spec:
  namespace: "*"      # all namespaces
  pricingModel: on-demand
  currency: USD
status:
  totalHourlyCost: "4.82"
  totalMonthlyCost: "3518.60"
  topWorkloads:
    - name: ml-training
      namespace: data
      hourlyCost: "1.92"
    - name: api-gateway
      namespace: production
      hourlyCost: "0.48"
```

## Prometheus Metrics

| Metric | Type | Labels |
|--------|------|--------|
| `cost_sentinel_workload_hourly_cost` | Gauge | namespace, workload, kind |
| `cost_sentinel_workload_monthly_estimate` | Gauge | namespace, workload, kind |
| `cost_sentinel_cluster_total_hourly` | Gauge | — |
| `cost_sentinel_reconcile_errors_total` | Counter | controller |

## Required IAM Permissions (IRSA)

```json
{
  "Effect": "Allow",
  "Action": ["pricing:GetProducts"],
  "Resource": "*"
}
```

## Development

```bash
# Run locally against a cluster
make run

# Generate CRD manifests
make manifests

# Run tests
make test

# Build and push image
make docker-build docker-push IMG=<your-registry>/eks-cost-sentinel:latest
```

## License

Apache 2.0 — by [Goutham Annem](https://linkedin.com/in/goutham-annem)
