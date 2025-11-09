# VolSync Template

## Flux Kustomization

This requires `components` and `postBuild` configured on the Flux Kustomization

```yaml
apiVersion: kustomize.toolkit.fluxcd.io/v1
kind: Kustomization
metadata:
    name: &app plex
    namespace: flux-system
spec:
    # ...
    components:
        - ../../../../components/volsync
    # ...
    postBuild:
        substitute:
            APP: *app
            VOLSYNC_CAPACITY: 5Gi
            NAMESPACE: *namespace
```

## Required `postBuild` vars

- `APP`: The application name
- `VOLSYNC_CAPACITY`: The PVC size
- `NAMESPACE` The namespace for the resources

## Optional `postBuild` vars

- TBD
