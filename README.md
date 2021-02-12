translations-refresher
======================

Automatic rollout kubernetes deployments on Loco translations updates.

## Usage

Run the app locally
```bash
go run *.go -kubeconfig ~/.kube/config
```

Build docker image
```bash
docker build -t translations-refresher:test .
```

Install Helm chart
```bash
helm install translations-refresher deploy/helm
```

## Testing

1. Install Kind (Kubernetes IN Docker) https://kind.sigs.k8s.io/docs/user/quick-start/#installation

2. Launch the local cluster
```bash
kind create cluster --name translations --config test/cluster.yaml
```

3. Load the docker image inside the cluster
```bash
kind load docker-image translations-refresher:test --name translations
```

4. Install the chart
```bash
helm install translations-refresher deploy/helm \
    --set image.tag=test \
    --set env.LOCO_API_KEY_DOCUMENTS=<loco_api_key> \
    --set env.LOCO_API_KEY_CATALOG=<loco_api_key> \
    --set env.LOCO_API_KEY_EMAILS=<loco_api_key>
```

5. Install the test deployment
kubectl apply -f test/deployment.yaml

The refresher is now running in the cluster and should have mutated the test deployment.
```bash
kubectl describe deployment/test-app
```
