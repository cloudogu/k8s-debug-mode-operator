# Verwendung

Um einen Debug-Modus zu erstellen, wenden Sie die [CRD-Bibliothek](https://github.com/cloudogu/k8s-debug-mode-cr-lib), den [Operator](https://github.com/cloudogu/k8s-debug-mode-operator) und eine Debug-Mode-Custom-Resource im Cluster an.
Das Format der Custom Resource ist in der [CRD-Bibliothek](https://github.com/cloudogu/k8s-debug-mode-cr-lib/blob/develop/k8s/helm-crd/templates/debugmode-crd.yaml) beschrieben.
Alternativ kann der Debug-Mode über unser Premium-Admin-Dogu gestartet werden.

## Interne Prozesse

### Singleton

Es kann immer nur eine Debug-Mode-CR gleichzeitig aktiv sein. Dies wird durch die folgende Validierung sichergestellt:
// +kubebuilder:validation:XValidation:rule="self.metadata.name == 'name'"

### Reconciliation und Phasen

Die Reconciliation-Schleife verfolgt Änderungen am Log-Level und prüft zuerst, ob der DebugMode aktiv ist, indem sie sicherstellt, dass der `DeactivationTimeStamp` noch nicht überschritten wurde.
Anschließend prüft sie, ob alle Dogus und Komponenten das erforderliche Debug-Log-Level haben.
Danach wartet der Operator in der Phase `WaitForRollback`, bis der `DeactivationTimestamp` überschritten wurde.
Sobald der `DeactivationTimestamp` überschritten wurde, deaktiviert er den Debug-Mode, wechselt in die Phase `Rollback` und verfolgt, ob alle Dogus und Komponenten wieder ihre zuvor gesetzten Log-Level erhalten haben.
Am Ende wechselt er in die Phase `Completed`.

### Status

Die vorherigen Log-Level von Dogus und Komponenten werden in einer ConfigMap gespeichert.
Sie wird zur Wiederherstellung der Dogu- und Komponenten-Log-Level benötigt, nachdem die DebugMode-CR die Phase `Rollback` erreicht hat und sich damit im deaktivierenden Zustand befindet.
Sobald die DebugMode-CR die Phase `Completed` erreicht, wird diese ConfigMap gelöscht.
