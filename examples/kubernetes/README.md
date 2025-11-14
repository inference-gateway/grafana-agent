# Grafana Agent Kubernetes Example

This example demonstrates how to deploy the grafana-agent on Kubernetes using **k3d**, **ctlptl**, the **Prometheus Operator**, **Grafana Operator**, and the **Inference Gateway Operator**. The setup provides a complete, cloud-native monitoring stack with AI-powered Grafana dashboard automation.

## Architecture

The example includes:

- **Grafana Agent**: A2A server for automating Grafana dashboard operations
- **Grafana**: Visualization platform (managed by Grafana Operator)
- **Prometheus**: Time-series metrics database (managed by Prometheus Operator)
- **ServiceMonitors**: Declarative Prometheus scrape configuration
- **Demo OTEL Service**: Sample service generating OpenTelemetry metrics
- **Inference Gateway**: AI provider router (managed by Inference Gateway Operator)
- **k3d Cluster**: Lightweight Kubernetes cluster for local development
- **Local Registry**: For storing and serving container images

## Prerequisites

Install the following tools:

- [Docker](https://docs.docker.com/get-docker/) - Container runtime
- [k3d](https://k3d.io/#installation) - k3s in Docker
- [ctlptl](https://github.com/tilt-dev/ctlptl#how-do-i-install-it) - Declarative cluster management
- [kubectl](https://kubernetes.io/docs/tasks/tools/) - Kubernetes CLI
- [Task](https://taskfile.dev/installation/) - Task runner (optional but recommended)

### Quick Installation (macOS with Homebrew)

```bash
brew install docker k3d ctlptl kubectl go-task
```

### Quick Installation (Linux)

```bash
# k3d
curl -s https://raw.githubusercontent.com/k3d-io/k3d/main/install.sh | bash

# ctlptl
CTLPTL_VERSION="0.8.25"
curl -fsSL https://github.com/tilt-dev/ctlptl/releases/download/v${CTLPTL_VERSION}/ctlptl.${CTLPTL_VERSION}.linux.x86_64.tar.gz | \
  sudo tar -xzv -C /usr/local/bin ctlptl

# kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Task
sh -c "$(curl --location https://taskfile.dev/install.sh)" -- -d -b /usr/local/bin
```

## Quick Start

### 1. Configure Environment

```bash
# Copy the example environment file
cp .env.example .env

# Edit .env and add at least one LLM provider API key
nano .env
```

Add your API keys to `.env`:

```bash
# At least one of these is required:
DEEPSEEK_API_KEY=your-deepseek-key-here
GOOGLE_API_KEY=your-google-key-here
ANTHROPIC_API_KEY=your-anthropic-key-here
OPENAI_API_KEY=your-openai-key-here
```

### 2. Deploy Everything

Using Task (recommended):

```bash
# Complete setup in one command
task up
```

Or manually:

```bash
# Create cluster
task create-cluster

# Install operators
task install-operators

# Build and push images
task build-images

# Create secrets
task create-secrets

# Deploy all services
task deploy

# Wait for services to be ready
task wait-ready
```

### 3. Access Services

Once deployed, access the services at:

- **Grafana**: http://localhost:3000 (admin/admin)
- **Prometheus**: http://localhost:9090
- **Demo Service**: http://localhost:8082
- **Demo Service Metrics**: http://localhost:8082/metrics
- **Grafana Agent**: http://localhost:8080

## Usage

### Using the Grafana Agent

The Grafana Agent exposes an A2A (Agent-to-Agent) API that you can interact with using the CLI or API calls.

#### Option 1: Using the Inference Gateway CLI

```bash
# Install the CLI (if not already installed)
# See: https://github.com/inference-gateway/cli

# Set environment variables
export INFER_GATEWAY_URL=http://localhost:8080
export INFER_A2A_ENABLED=true
export INFER_A2A_AGENTS=http://localhost:8080

# Start interactive chat
infer chat
```

Then try prompts like:

```text
Create a dashboard called "System Overview" showing CPU and memory usage from the demo-service. Use the agent.
```

#### Option 2: Using curl (Direct A2A API)

```bash
# Submit a task
curl -X POST http://localhost:8080/tasks \
  -H "Content-Type: application/json" \
  -d '{
    "input": "Create a dashboard showing HTTP request rates and latencies from demo-service. Use the agent."
  }'

# Get task status (replace TASK_ID)
curl http://localhost:8080/tasks/TASK_ID
```

#### Option 3: Using kubectl port-forward

If you're having issues with NodePort access:

```bash
# Port forward Grafana Agent
kubectl port-forward -n grafana-agent svc/grafana-agent 8080:8080

# In another terminal, use the CLI or curl as above
```

### Example Prompts

Try these example prompts with the agent:

#### Basic Dashboard Creation

```text
Create a dashboard named "Demo Service Metrics" with panels showing:
1. HTTP request rate by endpoint
2. CPU usage percentage
3. Memory usage in bytes
Use the agent.
```

#### Advanced Dashboard

```text
Create a comprehensive monitoring dashboard with the following panels:
1. Request rate by status code (stacked area chart)
2. P95 and P99 latency (line chart)
3. Error rate (with red threshold above 10%)
4. Active connections (gauge)
5. Queue size (gauge)
Use data from demo-service in Prometheus. Use the agent.
```

#### Modify Existing Dashboard

```text
Add a panel to the "Demo OTEL Service Dashboard" showing the error rate over time. Use the agent.
```

## Taskfile Commands

The Taskfile provides convenient commands for managing the deployment:

```bash
# Cluster Management
task create-cluster      # Create k3d cluster with ctlptl
task delete-cluster      # Delete the cluster
task down               # Complete teardown

# Deployment
task up                 # Complete setup (create + deploy + wait)
task deploy             # Deploy all manifests
task clean              # Clean up resources

# Development
task build-images       # Build and push images to registry
task rebuild-agent      # Rebuild and redeploy grafana-agent
task restart            # Restart grafana-agent deployment

# Monitoring
task status             # Show status of all resources
task logs               # Show grafana-agent logs
task logs-gateway       # Show inference-gateway logs
task logs-grafana       # Show Grafana logs

# Troubleshooting
task port-forward-grafana     # Port forward Grafana
task port-forward-prometheus  # Port forward Prometheus

# Help
task help               # List all available tasks
```

## Architecture Details

### Operators Used

#### Prometheus Operator

The [Prometheus Operator](https://github.com/prometheus-operator/prometheus-operator) provides Kubernetes-native deployment and management of Prometheus:

- **Prometheus CR**: Defines the Prometheus instance as a StatefulSet
- **ServiceMonitor CR**: Declaratively configures Prometheus scrape targets
- **PodMonitor CR**: Monitors pods directly (not used in this example)
- Automatic target discovery and configuration

This example uses **ServiceMonitors** for all metrics collection instead of static Prometheus configuration, providing:
- Automatic service discovery
- Declarative scrape configuration
- Dynamic updates when services change
- Better integration with Kubernetes RBAC

#### Grafana Operator

The [Grafana Operator](https://github.com/grafana/grafana-operator) (v5.x) manages Grafana instances using Kubernetes CRDs:

- **Grafana CR**: Defines the Grafana instance
- **GrafanaDatasource CR**: Configures datasources (Prometheus)
- **GrafanaDashboard CR**: Manages dashboards declaratively

This example installs the operator using the official cluster-scoped installation:
```bash
kubectl create -f https://github.com/grafana/grafana-operator/releases/latest/download/kustomize-cluster_scoped.yaml
```

The operator is installed in the `grafana-operator-system` namespace and watches for Grafana resources cluster-wide.

**Alternative installation methods:**
- **Helm**: `helm upgrade -i grafana-operator oci://ghcr.io/grafana/helm-charts/grafana-operator --version v5.20.0`
- **Namespace-scoped**: For multi-tenant environments, use `kustomize-namespace_scoped.yaml` instead
- See the [official documentation](https://grafana.github.io/grafana-operator/docs/installation/) for more options

#### Inference Gateway Operator

The [Inference Gateway Operator](https://github.com/inference-gateway/operator) manages Inference Gateway instances:

- **Gateway CR**: Defines the gateway deployment, providers, and configuration
- Auto-scaling, resource management, and provider configuration
- Integrates with Kubernetes secrets for API keys

### Cluster Configuration

The `cluster.yaml` file defines:

- **k3d cluster** with port mappings for services
- **Local registry** at `localhost:5005` for image storage
- **NodePort mappings**:
  - 3000 → Grafana (30000)
  - 9090 → Prometheus (30090)
  - 8082 → Demo Service (30082)
  - 8080 → Grafana Agent (30080)

### Networking

Services are exposed using:

- **NodePort**: For external access (mapped via k3d)
- **ClusterIP**: For internal service-to-service communication

## Demo Service Metrics

The demo OTEL service exports the following Prometheus metrics:

### Traditional Prometheus Metrics

- `http_requests_total` - Total HTTP requests by method, endpoint, and status
- `http_request_duration_seconds` - Request duration distribution (histogram)
- `active_connections` - Current number of active connections
- `processing_queue_size` - Current processing queue size
- `errors_total` - Total number of errors

### OpenTelemetry Metrics

- `cpu_usage_percent` - Simulated CPU usage percentage
- `memory_usage_bytes` - Simulated memory usage in bytes
- `request_latency_ms` - Request latency in milliseconds (histogram)

The service simulates realistic traffic patterns every 5 seconds.

## Troubleshooting

### Cluster Issues

**Problem**: Cluster fails to create

```bash
# Check Docker is running
docker ps

# Delete any existing cluster
task delete-cluster

# Try creating again
task create-cluster
```

**Problem**: Cannot access services via localhost

```bash
# Check port mappings
k3d cluster list
kubectl get svc -n grafana-agent

# Use port-forward as alternative
task port-forward-grafana
task port-forward-prometheus
```

### Operator Issues

**Problem**: Grafana Operator not installing

```bash
# Check operator status
kubectl get pods -n grafana-operator-system
kubectl logs -n grafana-operator-system deployment/grafana-operator-controller-manager

# Reinstall operator
kubectl delete -f https://github.com/grafana/grafana-operator/releases/latest/download/kustomize-cluster_scoped.yaml
kubectl create -f https://github.com/grafana/grafana-operator/releases/latest/download/kustomize-cluster_scoped.yaml
```

**Problem**: Inference Gateway Operator not ready

```bash
# Check operator status
kubectl get pods -n inference-gateway-system
kubectl logs -n inference-gateway-system deployment/operator

# Check CRDs are installed
kubectl get crd | grep gateway
```

### Service Issues

**Problem**: Grafana Agent cannot connect to Inference Gateway

```bash
# Check gateway status
kubectl get gateway -n grafana-agent
kubectl describe gateway inference-gateway -n grafana-agent

# Check gateway logs
task logs-gateway

# Check secrets are created
kubectl get secret api-keys -n grafana-agent
kubectl describe secret api-keys -n grafana-agent
```

**Problem**: No metrics in Prometheus

```bash
# Check Prometheus targets
# Open http://localhost:9090/targets

# Check demo-service is running
kubectl get pods -n grafana-agent
kubectl logs -n grafana-agent deployment/demo-service

# Check metrics endpoint directly
curl http://localhost:8082/metrics
```

**Problem**: Grafana not showing dashboards

```bash
# Check Grafana resources
kubectl get grafana,grafanadatasource,grafanadashboard -n grafana-agent

# Check Grafana logs
task logs-grafana

# Verify Grafana pod is running
kubectl get pods -n grafana-agent -l app=grafana
```

### Image Build Issues

**Problem**: Images not pulling/building

```bash
# Check registry is running
docker ps | grep registry

# Rebuild images
task build-images

# Check images are in registry
curl http://localhost:5005/v2/_catalog
```

**Problem**: Image pull errors in pods

```bash
# Check pod events
kubectl describe pod <pod-name> -n grafana-agent

# Ensure images are pushed
docker images | grep localhost:5005

# Try rebuilding
task rebuild-agent
```

### API Key Issues

**Problem**: Agent cannot access LLM providers

```bash
# Check if secrets exist
kubectl get secret api-keys -n grafana-agent -o yaml

# Update secrets
kubectl edit secret api-keys -n grafana-agent

# Or recreate from .env
kubectl delete secret api-keys -n grafana-agent
task create-secrets
```

## Development Workflow

### Modifying the Grafana Agent

1. Make changes to the agent code in the root directory
2. Rebuild and redeploy:

```bash
task rebuild-agent
```

3. Check logs:

```bash
task logs
```

### Adding Custom Metrics to Demo Service

1. Edit `../../examples/docker-compose/demo-service/main.go`
2. Rebuild the demo service image:

```bash
task build-images
kubectl rollout restart deployment/demo-service -n grafana-agent
```

### Creating Custom Dashboards

You can create dashboards manually in Grafana and export them as GrafanaDashboard CRs:

1. Create dashboard in Grafana UI
2. Export JSON via Dashboard Settings → JSON Model
3. Create a new manifest file:

```yaml
apiVersion: grafana.integreatly.org/v1beta1
kind: GrafanaDashboard
metadata:
  name: my-custom-dashboard
  namespace: grafana-agent
spec:
  instanceSelector:
    matchLabels:
      dashboards: "grafana"
  json: |
    <paste JSON here>
```

4. Apply the manifest:

```bash
kubectl apply -f manifests/my-custom-dashboard.yaml
```

## Clean Up

### Remove Resources (Keep Cluster)

```bash
task clean
```

### Delete Everything

```bash
task down
```

This will:
- Delete the k3d cluster
- Remove the local registry
- Clean up all resources

## Additional Resources

- [Grafana Operator Documentation](https://github.com/grafana/grafana-operator)
- [Inference Gateway Operator Documentation](https://github.com/inference-gateway/operator)
- [k3d Documentation](https://k3d.io/)
- [ctlptl Documentation](https://github.com/tilt-dev/ctlptl)
- [A2A Protocol Specification](https://github.com/inference-gateway/adk)
- [OpenTelemetry Documentation](https://opentelemetry.io/docs/)

## Contributing

If you find issues or have improvements for this example, please open an issue or pull request in the main repository.

## License

This example is part of the grafana-agent project and follows the same license.
