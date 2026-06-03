# cloud-native-reference-app

Happy-path cloud native reference app in Go with:

- Stateless todo REST API (`net/http`)
- Containerization (Docker)
- Kubernetes deployment (Helm)
- Health checks (`/healthz`, `/readyz`)
- Observability (OpenTelemetry traces + metrics)
- Autoscaling (HPA) with k6 load test
- CI with GitHub Actions + kubeconform

## Architecture

- `cmd/todo`: service entrypoint and config
- `internal/store`: thread-safe in-memory todo store
- `internal/api`: CRUD HTTP handlers and middleware
- `internal/health`: liveness/readiness handlers
- `internal/telemetry`: OTel setup + request instrumentation
- `deploy/helm/todo`: Kubernetes deployment chart
- `observability/otel-collector.yaml`: local OTel collector config
- `loadtest/k6-script.js`: load generation for HPA testing

## Prerequisites

Install:

- Go 1.22+
- Docker
- kubectl
- kind or minikube
- Helm 3+
- k6

Helpful links:

- Go: https://go.dev/dl/
- Docker: https://docs.docker.com/get-docker/
- kubectl: https://kubernetes.io/docs/tasks/tools/
- kind: https://kind.sigs.k8s.io/
- minikube: https://minikube.sigs.k8s.io/docs/start/
- Helm: https://helm.sh/docs/intro/install/
- k6: https://grafana.com/docs/k6/latest/set-up/install-k6/

## Run locally (without Kubernetes)

```bash
go test ./...
go run ./cmd/todo
```

In another terminal:

```bash
# health checks
curl -i http://localhost:8080/healthz
curl -i http://localhost:8080/readyz

# create todo
curl -i -X POST http://localhost:8080/todos \
  -H 'Content-Type: application/json' \
  -d '{"title":"learn cloud native"}'

# list todos
curl -i http://localhost:8080/todos

# get one todo
curl -i http://localhost:8080/todos/1

# update todo
curl -i -X PUT http://localhost:8080/todos/1 \
  -H 'Content-Type: application/json' \
  -d '{"title":"learn OTel","completed":true}'

# delete todo
curl -i -X DELETE http://localhost:8080/todos/1
```

## Build and run container

```bash
docker build -t todo-api:local .
docker run --rm -p 8080:8080 todo-api:local
```

Test endpoints the same way as above (`curl ...`).

## Deploy to local Kubernetes

### Option A: kind

```bash
kind create cluster --name todo-demo

docker build -t todo-api:local .
kind load docker-image todo-api:local --name todo-demo

helm install todo ./deploy/helm/todo \
  --set image.repository=todo-api \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent

kubectl get pods,svc,hpa
kubectl port-forward svc/todo-todo 8080:80
```

### Option B: minikube

```bash
minikube start

docker build -t todo-api:local .
minikube image load todo-api:local

helm install todo ./deploy/helm/todo \
  --set image.repository=todo-api \
  --set image.tag=local \
  --set image.pullPolicy=IfNotPresent

kubectl get pods,svc,hpa
kubectl port-forward svc/todo-todo 8080:80
```

Then test with `curl` (same commands as local run).

## Test autoscaling (HPA)

In one terminal:

```bash
kubectl get hpa -w
```

In another terminal:

```bash
kubectl get pods -w
```

Run load:

```bash
k6 run loadtest/k6-script.js --env BASE_URL=http://localhost:8080
```

You should see HPA increase replicas when CPU utilization rises, then scale down when load stops.

## Test observability (OpenTelemetry)

Run collector:

```bash
docker run --rm -p 4317:4317 -p 4318:4318 -p 9464:9464 \
  -v "$(pwd)/observability/otel-collector.yaml:/etc/otelcol/config.yaml" \
  otel/opentelemetry-collector:0.101.0
```

Generate traffic (`curl` or k6) and verify:

- Traces/metrics in collector logs (debug exporter)
- Prometheus metrics endpoint available on `http://localhost:9464/metrics`

## Cleanup

```bash
helm uninstall todo
kind delete cluster --name todo-demo
# OR: minikube delete
```
