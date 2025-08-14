# Usage

To create a debug mode apply the [crd lib](https://github.com/cloudogu/k8s-debug-mode-cr-lib), [operator](https://github.com/cloudogu/k8s-debug-mode-operator) and a debug mode custom resource in the cluster.
See [crd lib](https://github.com/cloudogu/k8s-debug-mode-cr-lib/blob/develop/k8s/helm-crd/templates/debugmode-crd.yaml) for the custom resource format. 
Alternatively the debug-mode can be started through our premium admin dogu.

## Internal processes

### Singleton

There always will be only one debug-mode-cr active at a time, this has been done through the usage of the following
validation: // +kubebuilder:validation:XValidation:rule="self.metadata.name == 'name'"

### Reconciliation and Phases

The Reconciliation Loop tracks log level changes and 
first checks that the DebugMode is active by checking that the DeactivationTimeStamp has not passed yet. 
After which it checks that all Dogus and Components have the required debug log level.
Then the Operator waits in 'WaitForRollback' Phase until the  DeactivationTimestamp has been passed. 
Once the DecativationTimestamp has passed it deactivates the Debug-Mode and switch into the 'Rollback' Phase
and keeps track that all Dogus and Components have their previously set log levels back. 
At the end it then moves into the 'Completed' Phase.

### State

Previous Log Levels of Dogu and Components are stored inside a ConfigMap, 
required for restoration of dogu and component log levels, which is after the DebugMode-CR
reaches the Phase 'Rollback' and thus is in its deactivating state.
Once the DebugMode-CR reaches the Phase: 'Completed' this ConfigMap will be deleted.