# Nats Scaler Demo
Nats Scaler Demo is a Kubernetes operator designed to automatically scale Deployments based on the number of pending messages
in a NATS JetStream consumer. It periodically queries the NATS monitoring endpoint (/jsz) over HTTP and adjusts the replica count of a target Deployment according to user-defined thresholds.

The operator is built using Kubebuilder and introduces a custom resource definition (CRD) called ScalingRule, which allows you to declaratively configure:
- The NATS monitoring endpoint (e.g., http://nats:8222)
- Target stream and consumer names to monitor
- The Kubernetes Deployment to be scaled
- Thresholds for scaling up and scaling down based on the number of pending messages
- Minimum and maximum replica counts

This approach enables responsive and queue-aware auto-scaling of microservices consuming from NATS JetStream, without requiring any modification to your application code.

## Getting Started

### Prerequisites
- go version v1.24.0+
- docker version 17.03+.
- kubectl version v1.11.3+.
- Access to a Kubernetes v1.11.3+ cluster.

### Dependency Installation

Before proceeding, ensure you have the following tools installed:

```sh
go install sigs.k8s.io/kind@v0.22.0
go install sigs.k8s.io/kustomize/kustomize/v5@latest # (if you want to use kustomize to control the configuration)
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.14.0
brew install kubebuilder  # or install manually from https://book.kubebuilder.io/quick-start.html
```

### Run Locally

To run the operator outside the cluster (e.g., for development or debugging), you must have a running Kubernetes cluster
accessible via your local kubeconfig (e.g., ~/.kube/config).
```sh
# Install CRDs into the cluster
make install

# Run the controller locally
make run 
```


### Deploying Locally with Kind

To deploy the operator in a local Kind cluster:

```sh
# Create a local cluster
kind create cluster --name nats-demo

# Build and load the image into Kind
make docker-build IMG=nats-scaler:dev
kind load docker-image nats-scaler:dev --name nats-demo

# Install CRDs and deploy the controller
make install
make deploy IMG=nats-scaler:dev
```


### Applying a Sample ScalingRule
The ScalingRule CR defines how the operator reacts to NATS JetStream metrics.

To apply a sample:
`kubectl apply -f config/samples/scalingrule_v1alpha1_scalingrule.yaml`
Example:
```yaml
apiVersion: scaling.my.domain/v1
kind: ScalingRule
metadata:
  labels:
    app.kubernetes.io/name: nats-scaler
    app.kubernetes.io/managed-by: kustomize
  name: scalingrule-sample
spec:
  deploymentName: myapp
  namespace: default
  minReplicas: 1
  maxReplicas: 5
  natsMonitoringURL: http://localhost:8222
  streamName: ORDERS
  consumerName: orders-consumer
  scaleUpThreshold: 10
  scaleDownThreshold: 3
  pollIntervalSeconds: 10
  ```

* To apply a dummy deployment just for testing: `kubectl apply -f test/fixtures/deploy.yaml`


## Tests
To docker-based env test:
```sh
make docker-test
```
Or to run locally:
```sh
make test
```

### To Uninstall
**Delete the instances (CRs) from the cluster:**

```sh
kubectl delete -k config/samples/
```

**Delete the APIs(CRDs) from the cluster:**

```sh
make uninstall
```

**UnDeploy the controller from the cluster:**

```sh
make undeploy
```