# 🐛 Debugging

Below is a general guide on trying to debug an issue with an resource or application. For example, if a workload/resource is not showing up or a pod has started but in a `CrashLoopBackOff` or `Pending` state.

1. Start by checking all Flux Kustomizations & Git Repository & OCI Repository and verify they are healthy.

    ```sh
    flux get sources oci -A
    flux get sources git -A
    flux get ks -A
    ```

2. Then check all the Flux Helm Releases and verify they are healthy.

    ```sh
    flux get hr -A
    ```

3. Then check the if the pod is present.

    ```sh
    kubectl -n <namespace> get pods -o wide
    ```

4. Then check the logs of the pod if its there.

    ```sh
    kubectl -n <namespace> logs <pod-name> -f
    # or
    stern -n <namespace> <fuzzy-name>
    ```

5. If a resource exists try to describe it to see what problems it might have.

    ```sh
    kubectl -n <namespace> describe <resource> <name>
    ```

6. Check the namespace events

    ```sh
    kubectl -n <namespace> get events --sort-by='.metadata.creationTimestamp'
    ```

## Nuking longhorn

say you messed up installing longhorn for some reason and you want to remove its CRDs:

```sh
for crd in $(kubectl get crd -o jsonpath={.items[*].metadata.name} | tr ' ' '\n' | grep longhorn.io); do   kubectl -n ${NAMESPACE} get $crd -o yaml | sed "s/\- longhorn.io//g" | kubectl apply -f -;   kubectl -n ${NAMESPACE} delete $crd --all;   kubectl delete crd/$crd; done
```

you should also remove the webhooks to leave your cluster in a usable state:
```sh
kubectl delete mutatingwebhookconfigurations longhorn-webhook-mutator
kubectl delete validatingwebhookconfigurations longhorn-webhook-validator
```
