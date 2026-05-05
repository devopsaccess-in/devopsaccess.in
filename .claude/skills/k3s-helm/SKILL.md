---
name: k3s-helm
description: Use this skill whenever the user asks to write, review, or modify Kubernetes manifests, Helm charts, or ArgoCD Application definitions for DevOps Access. Covers multi-tenant namespace patterns, resource limits, security contexts, and network policies. Triggers on phrases like "Kubernetes manifest", "Helm chart", "ArgoCD Application", "deploy to k3s", "namespace isolation", "pod security".
---

# DevOps Access Kubernetes + Helm Conventions

## Target environment

- Runtime: k3s v1.31+ on Hetzner CAX23 nodes
- CNI: Cilium (L3/L4/L7 policies)
- Policy: Kyverno for admission policies
- GitOps: ArgoCD with Application-of-Applications pattern
- Ingress: Traefik (bundled with k3s) — do NOT disable it

## Standard Helm chart structure
infra/helm/<service-name>/
├── Chart.yaml
├── values.yaml               # defaults
├── values-prod.yaml          # prod overrides
├── values-staging.yaml       # staging overrides
└── templates/
├── _helpers.tpl
├── serviceaccount.yaml
├── deployment.yaml
├── service.yaml
├── ingress.yaml
├── configmap.yaml
├── hpa.yaml
├── pdb.yaml
├── networkpolicy.yaml
└── servicemonitor.yaml
## Required in every Deployment

```yaml
spec:
template:
spec:
securityContext:
runAsNonRoot: true
runAsUser: 65532
fsGroup: 65532
seccompProfile:
type: RuntimeDefault
containers:
- name: {{ .Chart.Name }}
securityContext:
allowPrivilegeEscalation: false
readOnlyRootFilesystem: true
capabilities:
drop: ["ALL"]
resources:
requests:
cpu: "100m"
memory: "128Mi"
limits:
cpu: "1000m"
memory: "512Mi"
livenessProbe:
httpGet:
path: /healthz
port: 8080
initialDelaySeconds: 10
periodSeconds: 10
readinessProbe:
httpGet:
path: /ready
port: 8080
initialDelaySeconds: 5
periodSeconds: 5
## Multi-tenancy: namespace-per-tenant

Free-tier tenants get a namespace `tenant-<uuid>`. Isolation layers:

1. **NetworkPolicy default-deny** with explicit egress allow-list
2. **ResourceQuota + LimitRange** per-namespace caps
3. **Kyverno policies** blocking hostPath, privileged, forbidden registries
4. **Cilium L7 policies** blocking lateral movement between tenants

Namespace template:
```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: tenant-{{ .tenantId }}
  labels:
    tier: free
    tenant-id: {{ .tenantId }}
    managed-by: devopsaccess-control-plane
```

## ArgoCD Application pattern

One Application per (service, environment) combination:

```yaml
apiVersion: argoproj.io/v1alpha1
kind: Application
metadata:
  name: api-prod
  namespace: argocd
spec:
  project: devopsaccess-prod
  source:
    repoURL: https://github.com/<org>/devopsaccess.git
    targetRevision: main
    path: infra/helm/api
    helm:
      valueFiles:
        - values.yaml
        - values-prod.yaml
  destination:
    server: https://kubernetes.default.svc
    namespace: devopsaccess-prod
  syncPolicy:
    automated:
      prune: true
      selfHeal: true
    syncOptions:
      - CreateNamespace=true
      - ServerSideApply=true
```

## Hard rules

1. Every pod MUST have resource requests AND limits. No exceptions.
2. Every pod MUST run as non-root (`runAsNonRoot: true`, numeric UID).
3. No `:latest` tags. Pin to SHA or exact version.
4. Every Deployment MUST have a PodDisruptionBudget and HorizontalPodAutoscaler.
5. Every Service MUST have a ServiceMonitor for Prometheus scraping.
6. Never commit raw Secret manifests. Use SealedSecrets or External Secrets Operator.
7. Run `kubectl apply --dry-run=server` locally before committing manifest changes.

## Common pitfalls

- Traefik in k3s ignores Ingress `class` field; use Traefik annotations instead.
- `readOnlyRootFilesystem: true` breaks apps that write logs to `/tmp`. Add an `emptyDir` volume mounted at `/tmp`.
- HPA is silently ignored if no resource requests are set. Always set requests.
- Kyverno policies with `validationFailureAction: Enforce` will block admission. Test with `Audit` first.
- `kubectl apply` is denied by project settings — commit to git and let ArgoCD sync. Use `kubectl apply --dry-run=server` for validation.
