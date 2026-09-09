# Kopiur Template

Add `../../../../components/kopiur/hot` to a Flux Kustomization that manages an
application PVC named after `APP`.

Required post-build variables:

- `APP`: application and PVC name
- `NAMESPACE`: application namespace
- `KOPIUR_CAPACITY`: PVC size

The component creates the PVC, a Kopiur `SnapshotPolicy`, and an hourly
`SnapshotSchedule` using the shared `ClusterRepository` named `kopiur`.
