# Broker statefulset

```yaml
apiVersion: apps/v1
kind: StatefulSet
metadata:
  name: broker
  namespace: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
  uid: d011adea-7b89-4458-b46c-74a6bd4b40ca
  resourceVersion: '415250260'
  generation: 38
  creationTimestamp: '2025-05-13T15:29:19Z'
  labels:
    app.hivemq.cloud/team: platform
    app.kubernetes.io/instance: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: hivemq-broker
    app.kubernetes.io/version: 8.8.0
    argocd.argoproj.io/instance: apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
    helm.sh/chart: managed-hive-8.8.0
    hivemq-platform: broker
  annotations:
    javaoperatorsdk.io/previous: f247d71d-c052-43c7-b30a-e1fc8dca54ab,415249893
    kubectl.kubernetes.io/last-applied-configuration: >
      {"apiVersion":"hivemq.com/v1","kind":"HiveMQPlatform","metadata":{"annotations":{},"labels":{"app.hivemq.cloud/team":"platform","app.kubernetes.io/instance":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"hivemq-broker","app.kubernetes.io/version":"8.8.0","argocd.argoproj.io/instance":"apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","helm.sh/chart":"managed-hive-8.8.0"},"name":"broker","namespace":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"extensions":[{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-allow-all-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-amazon-kinesis-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-cloud-metering-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-enterprise-security-extension","secretName":"extension-config-hivemq-enterprise-security-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-kafka-extension","supportsHotReload":false}],"healthApiPort":9090,"metricsPath":"/","metricsPort":9399,"operatorRestApiPort":7979,"secretName":"broker-config","services":[{"metadata":{"name":"hivemq-broker-mqtts-probe"},"spec":{"clusterIP":"None","ports":[{"name":"mqtts-probe","port":1337,"targetPort":"mqtts-probe"}]}},{"metadata":{"name":"hivemq-broker-mqtt"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-0","port":8883,"targetPort":"mqtt"}]}},{"metadata":{"name":"hivemq-broker-mqtts-1"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-1","port":8884,"targetPort":"mqtts-1"}]}},{"metadata":{"name":"hivemq-broker-ws-0"},"spec":{"clusterIP":"None","ports":[{"name":"ws-0","port":5883,"targetPort":"ws-0"}]}},{"metadata":{"name":"hivemq-broker-ws-1"},"spec":{"clusterIP":"None","ports":[{"name":"ws-1","port":5884,"targetPort":"ws-1"}]}},{"metadata":{"name":"hivemq-broker-cc"},"spec":{"clusterIP":"None","ports":[{"name":"cc","port":8080,"targetPort":"cc"}]}},{"metadata":{"name":"hivemq-broker-api"},"spec":{"clusterIP":"None","ports":[{"name":"api","port":8081,"targetPort":"api"}]}},{"metadata":{"name":"metrics-0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"ports":[{"name":"metrics","port":9399,"targetPort":"metrics"}]}}],"statefulSet":{"spec":{"replicas":2,"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"designation","operator":"In","values":["tier1"]}]}]}},"podAntiAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"podAffinityTerm":{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/name","operator":"In","values":["hivemq-broker"]}]},"topologyKey":"topology.kubernetes.io/zone"},"weight":100}],"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/instance","operator":"In","values":["0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"]}]},"topologyKey":"kubernetes.io/hostname"}]}},"containers":[{"env":[{"name":"JAVA_OPTS","value":"-XX:+UnlockExperimentalVMOptions
      -XX:InitialRAMPercentage=50
      -XX:MaxRAMPercentage=50"},{"name":"HIVEID","value":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},{"name":"ESE_DATABASE_NAME","valueFrom":{"secretKeyRef":{"key":"dbname","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_PASSWORD","valueFrom":{"secretKeyRef":{"key":"password","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_USER","valueFrom":{"secretKeyRef":{"key":"username","name":"pguser-hivemq"}}},{"name":"HIVEMQ_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HIVEMQ_INTERNAL_ANALYTIC_METRICS","value":"true"},{"name":"HIVEMQ_INTERNAL_NORMALIZED_MESSAGE_SIZE_IN_BYTES","value":"5120"},{"name":"HIVEMQ_LOGBACK_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HMQC_METERING_NORMALIZED_MESSAGE_BYTES","value":"5120"},{"name":"HMQC_METERING_PROBE_TOPIC_PREFIX","value":"probes"},{"name":"OAUTH_SECRET_KEY","valueFrom":{"secretKeyRef":{"key":"secret-key","name":"oauth-secrets"}}}],"image":"registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64","imagePullPolicy":"IfNotPresent","name":"hivemq","ports":[{"containerPort":1337,"name":"mqtts-probe"},{"containerPort":8883,"name":"mqtt"},{"containerPort":8884,"name":"mqtts-1"},{"containerPort":5883,"name":"ws-0"},{"containerPort":5884,"name":"ws-1"},{"containerPort":8080,"name":"cc"},{"containerPort":8081,"name":"api"},{"containerPort":9090,"name":"health"},{"containerPort":9399,"name":"metrics"}],"resources":{"limits":{"cpu":"4000m","memory":"3072M"},"requests":{"cpu":"400m","memory":"2048M"}},"volumeMounts":[{"mountPath":"/opt/hivemq/data","name":"data"},{"mountPath":"/opt/hivemq/log","name":"logs"},{"mountPath":"/opt/hivemq/license","name":"licenses"},{"mountPath":"/opt/hivemq/conf/cluster-transport-keystore","name":"broker-cluster-transport-tls","readOnly":true},{"mountPath":"/opt/hivemq/conf/tls","name":"hive-certificates","readOnly":true}]}],"imagePullSecrets":[{"name":"harbor-pull-secret"}],"securityContext":{"fsGroup":10000,"fsGroupChangePolicy":"OnRootMismatch"},"tolerations":[{"effect":"NoSchedule","key":"designation","operator":"Equal","value":"tier1"}],"volumes":[{"emptyDir":{},"name":"logs"},{"name":"licenses","secret":{"secretName":"hivemq-common-licenses"}},{"name":"broker-cluster-transport-tls","secret":{"secretName":"broker-cluster-transport-tls"}},{"name":"hive-certificates","secret":{"secretName":"hive-certificates"}}]}},"volumeClaimTemplates":[{"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"creationTimestamp":null,"name":"data"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"5Gi"}},"storageClassName":"broker-standard-1","volumeMode":"Filesystem"}}]}}}}
  ownerReferences:
    - apiVersion: hivemq.com/v1
      kind: HiveMQPlatform
      name: broker
      uid: 7130cf4b-c416-4e4c-a4e2-c27b7c24d843
  selfLink: >-
    /apis/apps/v1/namespaces/0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8/statefulsets/broker
status:
  observedGeneration: 38
  replicas: 2
  readyReplicas: 2
  updatedReplicas: 2
  currentRevision: broker-869db4bfc8
  updateRevision: broker-64b5c4cb99
  collisionCount: 0
  availableReplicas: 2
spec:
  replicas: 2
  selector:
    matchLabels:
      hivemq-platform: broker
  template:
    metadata:
      creationTimestamp: null
      labels:
        app.hivemq.cloud/team: platform
        app.kubernetes.io/instance: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
        app.kubernetes.io/managed-by: Helm
        app.kubernetes.io/name: hivemq-broker
        app.kubernetes.io/version: 8.8.0
        argocd.argoproj.io/instance: apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
        helm.sh/chart: managed-hive-8.8.0
        hivemq-platform: broker
      annotations:
        kubectl.kubernetes.io/last-applied-configuration: >
          {"apiVersion":"hivemq.com/v1","kind":"HiveMQPlatform","metadata":{"annotations":{},"labels":{"app.hivemq.cloud/team":"platform","app.kubernetes.io/instance":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"hivemq-broker","app.kubernetes.io/version":"8.8.0","argocd.argoproj.io/instance":"apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","helm.sh/chart":"managed-hive-8.8.0"},"name":"broker","namespace":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"extensions":[{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-allow-all-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-amazon-kinesis-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-cloud-metering-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-enterprise-security-extension","secretName":"extension-config-hivemq-enterprise-security-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-kafka-extension","supportsHotReload":false}],"healthApiPort":9090,"metricsPath":"/","metricsPort":9399,"operatorRestApiPort":7979,"secretName":"broker-config","services":[{"metadata":{"name":"hivemq-broker-mqtts-probe"},"spec":{"clusterIP":"None","ports":[{"name":"mqtts-probe","port":1337,"targetPort":"mqtts-probe"}]}},{"metadata":{"name":"hivemq-broker-mqtt"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-0","port":8883,"targetPort":"mqtt"}]}},{"metadata":{"name":"hivemq-broker-mqtts-1"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-1","port":8884,"targetPort":"mqtts-1"}]}},{"metadata":{"name":"hivemq-broker-ws-0"},"spec":{"clusterIP":"None","ports":[{"name":"ws-0","port":5883,"targetPort":"ws-0"}]}},{"metadata":{"name":"hivemq-broker-ws-1"},"spec":{"clusterIP":"None","ports":[{"name":"ws-1","port":5884,"targetPort":"ws-1"}]}},{"metadata":{"name":"hivemq-broker-cc"},"spec":{"clusterIP":"None","ports":[{"name":"cc","port":8080,"targetPort":"cc"}]}},{"metadata":{"name":"hivemq-broker-api"},"spec":{"clusterIP":"None","ports":[{"name":"api","port":8081,"targetPort":"api"}]}},{"metadata":{"name":"metrics-0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"ports":[{"name":"metrics","port":9399,"targetPort":"metrics"}]}}],"statefulSet":{"spec":{"replicas":2,"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"designation","operator":"In","values":["tier1"]}]}]}},"podAntiAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"podAffinityTerm":{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/name","operator":"In","values":["hivemq-broker"]}]},"topologyKey":"topology.kubernetes.io/zone"},"weight":100}],"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/instance","operator":"In","values":["0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"]}]},"topologyKey":"kubernetes.io/hostname"}]}},"containers":[{"env":[{"name":"JAVA_OPTS","value":"-XX:+UnlockExperimentalVMOptions
          -XX:InitialRAMPercentage=50
          -XX:MaxRAMPercentage=50"},{"name":"HIVEID","value":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},{"name":"ESE_DATABASE_NAME","valueFrom":{"secretKeyRef":{"key":"dbname","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_PASSWORD","valueFrom":{"secretKeyRef":{"key":"password","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_USER","valueFrom":{"secretKeyRef":{"key":"username","name":"pguser-hivemq"}}},{"name":"HIVEMQ_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HIVEMQ_INTERNAL_ANALYTIC_METRICS","value":"true"},{"name":"HIVEMQ_INTERNAL_NORMALIZED_MESSAGE_SIZE_IN_BYTES","value":"5120"},{"name":"HIVEMQ_LOGBACK_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HMQC_METERING_NORMALIZED_MESSAGE_BYTES","value":"5120"},{"name":"HMQC_METERING_PROBE_TOPIC_PREFIX","value":"probes"},{"name":"OAUTH_SECRET_KEY","valueFrom":{"secretKeyRef":{"key":"secret-key","name":"oauth-secrets"}}}],"image":"registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64","imagePullPolicy":"IfNotPresent","name":"hivemq","ports":[{"containerPort":1337,"name":"mqtts-probe"},{"containerPort":8883,"name":"mqtt"},{"containerPort":8884,"name":"mqtts-1"},{"containerPort":5883,"name":"ws-0"},{"containerPort":5884,"name":"ws-1"},{"containerPort":8080,"name":"cc"},{"containerPort":8081,"name":"api"},{"containerPort":9090,"name":"health"},{"containerPort":9399,"name":"metrics"}],"resources":{"limits":{"cpu":"4000m","memory":"3072M"},"requests":{"cpu":"400m","memory":"2048M"}},"volumeMounts":[{"mountPath":"/opt/hivemq/data","name":"data"},{"mountPath":"/opt/hivemq/log","name":"logs"},{"mountPath":"/opt/hivemq/license","name":"licenses"},{"mountPath":"/opt/hivemq/conf/cluster-transport-keystore","name":"broker-cluster-transport-tls","readOnly":true},{"mountPath":"/opt/hivemq/conf/tls","name":"hive-certificates","readOnly":true}]}],"imagePullSecrets":[{"name":"harbor-pull-secret"}],"securityContext":{"fsGroup":10000,"fsGroupChangePolicy":"OnRootMismatch"},"tolerations":[{"effect":"NoSchedule","key":"designation","operator":"Equal","value":"tier1"}],"volumes":[{"emptyDir":{},"name":"logs"},{"name":"licenses","secret":{"secretName":"hivemq-common-licenses"}},{"name":"broker-cluster-transport-tls","secret":{"secretName":"broker-cluster-transport-tls"}},{"name":"hive-certificates","secret":{"secretName":"hive-certificates"}}]}},"volumeClaimTemplates":[{"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"creationTimestamp":null,"name":"data"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"5Gi"}},"storageClassName":"broker-standard-1","volumeMode":"Filesystem"}}]}}}}
        kubernetes-resource-versions: >-
          {env-var-secret-oauth-secrets=375386304,
          broker-configuration-v2=1977827560,
          env-var-secret-pguser-hivemq=151919322}
    spec:
      volumes:
        - name: logs
          emptyDir: {}
        - name: licenses
          secret:
            secretName: hivemq-common-licenses
            defaultMode: 420
        - name: broker-cluster-transport-tls
          secret:
            secretName: broker-cluster-transport-tls
            defaultMode: 420
        - name: hive-certificates
          secret:
            secretName: hive-certificates
            defaultMode: 420
        - name: pod-info
          configMap:
            name: hivemq-platform-broker-dynamic-state
            defaultMode: 420
        - name: extension-configuration-hivemq-enterprise-security-extension
          secret:
            secretName: extension-config-hivemq-enterprise-security-extension
            defaultMode: 420
        - name: broker-configuration
          secret:
            secretName: broker-config
            defaultMode: 420
        - name: operator-init
          emptyDir: {}
      initContainers:
        - name: hivemq-platform-operator-init
          image: docker.io/hivemq/hivemq-platform-operator-init:1.7.0
          resources:
            limits:
              cpu: 250m
              ephemeral-storage: 1Gi
              memory: 100Mi
            requests:
              cpu: 250m
              ephemeral-storage: 1Gi
              memory: 100Mi
          volumeMounts:
            - name: operator-init
              mountPath: /hivemq
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
          imagePullPolicy: IfNotPresent
      containers:
        - name: hivemq
          image: >-
            registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64
          command:
            - /opt/hivemq/operator/bin/run.sh
          ports:
            - name: mqtts-probe
              containerPort: 1337
              protocol: TCP
            - name: mqtt
              containerPort: 8883
              protocol: TCP
            - name: mqtts-1
              containerPort: 8884
              protocol: TCP
            - name: ws-0
              containerPort: 5883
              protocol: TCP
            - name: ws-1
              containerPort: 5884
              protocol: TCP
            - name: cc
              containerPort: 8080
              protocol: TCP
            - name: api
              containerPort: 8081
              protocol: TCP
            - name: health
              containerPort: 9090
              protocol: TCP
            - name: metrics
              containerPort: 9399
              protocol: TCP
          env:
            - name: JAVA_OPTS
              value: >-
                -XX:+UnlockExperimentalVMOptions -XX:InitialRAMPercentage=50
                -XX:MaxRAMPercentage=50
            - name: HIVEID
              value: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
            - name: ESE_DATABASE_NAME
              valueFrom:
                secretKeyRef:
                  name: pguser-hivemq
                  key: dbname
            - name: ESE_DATABASE_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: pguser-hivemq
                  key: password
            - name: ESE_DATABASE_USER
              valueFrom:
                secretKeyRef:
                  name: pguser-hivemq
                  key: username
            - name: HIVEMQ_CONFIG_FOLDER
              value: /opt/hivemq/conf-k8s
            - name: HIVEMQ_INTERNAL_ANALYTIC_METRICS
              value: 'true'
            - name: HIVEMQ_INTERNAL_NORMALIZED_MESSAGE_SIZE_IN_BYTES
              value: '5120'
            - name: HIVEMQ_LOGBACK_CONFIG_FOLDER
              value: /opt/hivemq/conf-k8s
            - name: HMQC_METERING_NORMALIZED_MESSAGE_BYTES
              value: '5120'
            - name: HMQC_METERING_PROBE_TOPIC_PREFIX
              value: probes
            - name: OAUTH_SECRET_KEY
              valueFrom:
                secretKeyRef:
                  name: oauth-secrets
                  key: secret-key
            - name: HIVEMQ_BIND_ADDRESS
              valueFrom:
                fieldRef:
                  apiVersion: v1
                  fieldPath: status.podIP
          resources:
            limits:
              cpu: '4'
              memory: 3072M
            requests:
              cpu: 400m
              memory: 2048M
          volumeMounts:
            - name: data
              mountPath: /opt/hivemq/data
            - name: logs
              mountPath: /opt/hivemq/log
            - name: licenses
              mountPath: /opt/hivemq/license
            - name: broker-cluster-transport-tls
              readOnly: true
              mountPath: /opt/hivemq/conf/cluster-transport-keystore
            - name: hive-certificates
              readOnly: true
              mountPath: /opt/hivemq/conf/tls
            - name: broker-configuration
              mountPath: /opt/hivemq/conf-k8s/
            - name: pod-info
              mountPath: /etc/podinfo/
            - name: extension-configuration-hivemq-enterprise-security-extension
              mountPath: >-
                /opt/hivemq/extensions/hivemq-enterprise-security-extension/conf/
            - name: operator-init
              mountPath: /opt/hivemq/operator/
          livenessProbe:
            httpGet:
              path: /liveness
              port: 7979
              scheme: HTTP
            initialDelaySeconds: 15
            timeoutSeconds: 1
            periodSeconds: 30
            successThreshold: 1
            failureThreshold: 240
          readinessProbe:
            httpGet:
              path: /readiness
              port: 7979
              scheme: HTTP
            initialDelaySeconds: 3
            timeoutSeconds: 1
            periodSeconds: 5
            successThreshold: 1
            failureThreshold: 3
          terminationMessagePath: /dev/termination-log
          terminationMessagePolicy: File
          imagePullPolicy: IfNotPresent
      restartPolicy: Always
      terminationGracePeriodSeconds: 3600
      dnsPolicy: ClusterFirst
      serviceAccountName: hivemq-platform-pod-broker
      serviceAccount: hivemq-platform-pod-broker
      securityContext:
        fsGroup: 10000
        fsGroupChangePolicy: OnRootMismatch
      imagePullSecrets:
        - name: harbor-pull-secret
      affinity:
        nodeAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            nodeSelectorTerms:
              - matchExpressions:
                  - key: designation
                    operator: In
                    values:
                      - tier1
        podAntiAffinity:
          requiredDuringSchedulingIgnoredDuringExecution:
            - labelSelector:
                matchExpressions:
                  - key: app.kubernetes.io/instance
                    operator: In
                    values:
                      - 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
              topologyKey: kubernetes.io/hostname
          preferredDuringSchedulingIgnoredDuringExecution:
            - weight: 100
              podAffinityTerm:
                labelSelector:
                  matchExpressions:
                    - key: app.kubernetes.io/name
                      operator: In
                      values:
                        - hivemq-broker
                topologyKey: topology.kubernetes.io/zone
      schedulerName: default-scheduler
      tolerations:
        - key: designation
          operator: Equal
          value: tier1
          effect: NoSchedule
  volumeClaimTemplates:
    - kind: PersistentVolumeClaim
      apiVersion: v1
      metadata:
        name: data
        creationTimestamp: null
      spec:
        accessModes:
          - ReadWriteOnce
        resources:
          requests:
            storage: 5Gi
        storageClassName: broker-standard-1
        volumeMode: Filesystem
      status:
        phase: Pending
  serviceName: hivemq-broker-cluster
  podManagementPolicy: OrderedReady
  updateStrategy:
    type: OnDelete
  revisionHistoryLimit: 10
  persistentVolumeClaimRetentionPolicy:
    whenDeleted: Retain
    whenScaled: Retain
```

# one of two broker pods
```yaml
apiVersion: v1
kind: Pod
metadata:
  name: broker-0
  generateName: broker-
  namespace: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
  uid: 51fe7f8e-e694-4aa1-a5c0-e1e6d74fbf90
  resourceVersion: '415249879'
  creationTimestamp: '2025-06-17T07:57:56Z'
  labels:
    app.hivemq.cloud/team: platform
    app.kubernetes.io/instance: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
    app.kubernetes.io/managed-by: Helm
    app.kubernetes.io/name: hivemq-broker
    app.kubernetes.io/version: 8.8.0
    apps.kubernetes.io/pod-index: '0'
    argocd.argoproj.io/instance: apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
    controller-revision-hash: broker-64b5c4cb99
    helm.sh/chart: managed-hive-8.8.0
    hivemq-platform: broker
    statefulset.kubernetes.io/pod-name: broker-0
  annotations:
    hivemq/platform-operator-init-app-version: 1.7.0
    kubectl.kubernetes.io/last-applied-configuration: >
      {"apiVersion":"hivemq.com/v1","kind":"HiveMQPlatform","metadata":{"annotations":{},"labels":{"app.hivemq.cloud/team":"platform","app.kubernetes.io/instance":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","app.kubernetes.io/managed-by":"Helm","app.kubernetes.io/name":"hivemq-broker","app.kubernetes.io/version":"8.8.0","argocd.argoproj.io/instance":"apiary-hives_0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8","helm.sh/chart":"managed-hive-8.8.0"},"name":"broker","namespace":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"extensions":[{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-allow-all-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-amazon-kinesis-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-cloud-metering-extension","supportsHotReload":false},{"enabled":true,"extensionUri":"preinstalled","id":"hivemq-enterprise-security-extension","secretName":"extension-config-hivemq-enterprise-security-extension","supportsHotReload":false},{"enabled":false,"extensionUri":"preinstalled","id":"hivemq-kafka-extension","supportsHotReload":false}],"healthApiPort":9090,"metricsPath":"/","metricsPort":9399,"operatorRestApiPort":7979,"secretName":"broker-config","services":[{"metadata":{"name":"hivemq-broker-mqtts-probe"},"spec":{"clusterIP":"None","ports":[{"name":"mqtts-probe","port":1337,"targetPort":"mqtts-probe"}]}},{"metadata":{"name":"hivemq-broker-mqtt"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-0","port":8883,"targetPort":"mqtt"}]}},{"metadata":{"name":"hivemq-broker-mqtts-1"},"spec":{"clusterIP":"None","ports":[{"name":"hivemq-broker-mqtts-1","port":8884,"targetPort":"mqtts-1"}]}},{"metadata":{"name":"hivemq-broker-ws-0"},"spec":{"clusterIP":"None","ports":[{"name":"ws-0","port":5883,"targetPort":"ws-0"}]}},{"metadata":{"name":"hivemq-broker-ws-1"},"spec":{"clusterIP":"None","ports":[{"name":"ws-1","port":5884,"targetPort":"ws-1"}]}},{"metadata":{"name":"hivemq-broker-cc"},"spec":{"clusterIP":"None","ports":[{"name":"cc","port":8080,"targetPort":"cc"}]}},{"metadata":{"name":"hivemq-broker-api"},"spec":{"clusterIP":"None","ports":[{"name":"api","port":8081,"targetPort":"api"}]}},{"metadata":{"name":"metrics-0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},"spec":{"ports":[{"name":"metrics","port":9399,"targetPort":"metrics"}]}}],"statefulSet":{"spec":{"replicas":2,"template":{"spec":{"affinity":{"nodeAffinity":{"requiredDuringSchedulingIgnoredDuringExecution":{"nodeSelectorTerms":[{"matchExpressions":[{"key":"designation","operator":"In","values":["tier1"]}]}]}},"podAntiAffinity":{"preferredDuringSchedulingIgnoredDuringExecution":[{"podAffinityTerm":{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/name","operator":"In","values":["hivemq-broker"]}]},"topologyKey":"topology.kubernetes.io/zone"},"weight":100}],"requiredDuringSchedulingIgnoredDuringExecution":[{"labelSelector":{"matchExpressions":[{"key":"app.kubernetes.io/instance","operator":"In","values":["0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"]}]},"topologyKey":"kubernetes.io/hostname"}]}},"containers":[{"env":[{"name":"JAVA_OPTS","value":"-XX:+UnlockExperimentalVMOptions
      -XX:InitialRAMPercentage=50
      -XX:MaxRAMPercentage=50"},{"name":"HIVEID","value":"0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8"},{"name":"ESE_DATABASE_NAME","valueFrom":{"secretKeyRef":{"key":"dbname","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_PASSWORD","valueFrom":{"secretKeyRef":{"key":"password","name":"pguser-hivemq"}}},{"name":"ESE_DATABASE_USER","valueFrom":{"secretKeyRef":{"key":"username","name":"pguser-hivemq"}}},{"name":"HIVEMQ_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HIVEMQ_INTERNAL_ANALYTIC_METRICS","value":"true"},{"name":"HIVEMQ_INTERNAL_NORMALIZED_MESSAGE_SIZE_IN_BYTES","value":"5120"},{"name":"HIVEMQ_LOGBACK_CONFIG_FOLDER","value":"/opt/hivemq/conf-k8s"},{"name":"HMQC_METERING_NORMALIZED_MESSAGE_BYTES","value":"5120"},{"name":"HMQC_METERING_PROBE_TOPIC_PREFIX","value":"probes"},{"name":"OAUTH_SECRET_KEY","valueFrom":{"secretKeyRef":{"key":"secret-key","name":"oauth-secrets"}}}],"image":"registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64","imagePullPolicy":"IfNotPresent","name":"hivemq","ports":[{"containerPort":1337,"name":"mqtts-probe"},{"containerPort":8883,"name":"mqtt"},{"containerPort":8884,"name":"mqtts-1"},{"containerPort":5883,"name":"ws-0"},{"containerPort":5884,"name":"ws-1"},{"containerPort":8080,"name":"cc"},{"containerPort":8081,"name":"api"},{"containerPort":9090,"name":"health"},{"containerPort":9399,"name":"metrics"}],"resources":{"limits":{"cpu":"4000m","memory":"3072M"},"requests":{"cpu":"400m","memory":"2048M"}},"volumeMounts":[{"mountPath":"/opt/hivemq/data","name":"data"},{"mountPath":"/opt/hivemq/log","name":"logs"},{"mountPath":"/opt/hivemq/license","name":"licenses"},{"mountPath":"/opt/hivemq/conf/cluster-transport-keystore","name":"broker-cluster-transport-tls","readOnly":true},{"mountPath":"/opt/hivemq/conf/tls","name":"hive-certificates","readOnly":true}]}],"imagePullSecrets":[{"name":"harbor-pull-secret"}],"securityContext":{"fsGroup":10000,"fsGroupChangePolicy":"OnRootMismatch"},"tolerations":[{"effect":"NoSchedule","key":"designation","operator":"Equal","value":"tier1"}],"volumes":[{"emptyDir":{},"name":"logs"},{"name":"licenses","secret":{"secretName":"hivemq-common-licenses"}},{"name":"broker-cluster-transport-tls","secret":{"secretName":"broker-cluster-transport-tls"}},{"name":"hive-certificates","secret":{"secretName":"hive-certificates"}}]}},"volumeClaimTemplates":[{"apiVersion":"v1","kind":"PersistentVolumeClaim","metadata":{"creationTimestamp":null,"name":"data"},"spec":{"accessModes":["ReadWriteOnce"],"resources":{"requests":{"storage":"5Gi"}},"storageClassName":"broker-standard-1","volumeMode":"Filesystem"}}]}}}}
    kubernetes-resource-versions: >-
      {env-var-secret-oauth-secrets=375386304,
      broker-configuration-v2=1977827560,
      env-var-secret-pguser-hivemq=151919322}
  ownerReferences:
    - apiVersion: apps/v1
      kind: StatefulSet
      name: broker
      uid: d011adea-7b89-4458-b46c-74a6bd4b40ca
      controller: true
      blockOwnerDeletion: true
  selfLink: /api/v1/namespaces/0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8/pods/broker-0
status:
  phase: Running
  conditions:
    - type: PodReadyToStartContainers
      status: 'True'
      lastProbeTime: null
      lastTransitionTime: '2025-06-17T07:57:58Z'
    - type: Initialized
      status: 'True'
      lastProbeTime: null
      lastTransitionTime: '2025-06-17T07:57:58Z'
    - type: Ready
      status: 'True'
      lastProbeTime: null
      lastTransitionTime: '2025-06-17T07:58:27Z'
    - type: ContainersReady
      status: 'True'
      lastProbeTime: null
      lastTransitionTime: '2025-06-17T07:58:27Z'
    - type: PodScheduled
      status: 'True'
      lastProbeTime: null
      lastTransitionTime: '2025-06-17T07:57:56Z'
  hostIP: 10.255.7.232
  hostIPs:
    - ip: 10.255.7.232
  podIP: 100.64.101.60
  podIPs:
    - ip: 100.64.101.60
  startTime: '2025-06-17T07:57:56Z'
  initContainerStatuses:
    - name: hivemq-platform-operator-init
      state:
        terminated:
          exitCode: 0
          reason: Completed
          startedAt: '2025-06-17T07:57:57Z'
          finishedAt: '2025-06-17T07:57:57Z'
          containerID: >-
            containerd://7767e58acb2ac0f87a3d0137becaaea26d29c35bf974a9e26a6194b9fb616a23
      lastState: {}
      ready: true
      restartCount: 0
      image: docker.io/hivemq/hivemq-platform-operator-init:1.7.0
      imageID: >-
        docker.io/hivemq/hivemq-platform-operator-init@sha256:47f8549c5d54abd12fdbafb3197bbb08d2c2d436c608500ceeca0f7b5f673209
      containerID: >-
        containerd://7767e58acb2ac0f87a3d0137becaaea26d29c35bf974a9e26a6194b9fb616a23
      started: false
      volumeMounts:
        - name: operator-init
          mountPath: /hivemq
        - name: kube-api-access-h9cqt
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
          readOnly: true
          recursiveReadOnly: Disabled
  containerStatuses:
    - name: hivemq
      state:
        running:
          startedAt: '2025-06-17T07:57:58Z'
      lastState: {}
      ready: true
      restartCount: 0
      image: >-
        registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64
      imageID: >-
        registry.hmqc.dev/hivemq-cloud/broker@sha256:37f425b2cb0b64ea0cfab90f19e3cf61cc05e95cf02b819d6055a4eb5718da1d
      containerID: >-
        containerd://f0e60926c35fd73f60032701daea3e8e93552a4d16af063cd48bdf7f9e26ce79
      started: true
      volumeMounts:
        - name: data
          mountPath: /opt/hivemq/data
        - name: logs
          mountPath: /opt/hivemq/log
        - name: licenses
          mountPath: /opt/hivemq/license
        - name: broker-cluster-transport-tls
          mountPath: /opt/hivemq/conf/cluster-transport-keystore
          readOnly: true
          recursiveReadOnly: Disabled
        - name: hive-certificates
          mountPath: /opt/hivemq/conf/tls
          readOnly: true
          recursiveReadOnly: Disabled
        - name: broker-configuration
          mountPath: /opt/hivemq/conf-k8s/
        - name: pod-info
          mountPath: /etc/podinfo/
        - name: extension-configuration-hivemq-enterprise-security-extension
          mountPath: /opt/hivemq/extensions/hivemq-enterprise-security-extension/conf/
        - name: operator-init
          mountPath: /opt/hivemq/operator/
        - name: kube-api-access-h9cqt
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
          readOnly: true
          recursiveReadOnly: Disabled
  qosClass: Burstable
spec:
  volumes:
    - name: data
      persistentVolumeClaim:
        claimName: data-broker-0
    - name: logs
      emptyDir: {}
    - name: licenses
      secret:
        secretName: hivemq-common-licenses
        defaultMode: 420
    - name: broker-cluster-transport-tls
      secret:
        secretName: broker-cluster-transport-tls
        defaultMode: 420
    - name: hive-certificates
      secret:
        secretName: hive-certificates
        defaultMode: 420
    - name: pod-info
      configMap:
        name: hivemq-platform-broker-dynamic-state
        defaultMode: 420
    - name: extension-configuration-hivemq-enterprise-security-extension
      secret:
        secretName: extension-config-hivemq-enterprise-security-extension
        defaultMode: 420
    - name: broker-configuration
      secret:
        secretName: broker-config
        defaultMode: 420
    - name: operator-init
      emptyDir: {}
    - name: kube-api-access-h9cqt
      projected:
        sources:
          - serviceAccountToken:
              expirationSeconds: 3607
              path: token
          - configMap:
              name: kube-root-ca.crt
              items:
                - key: ca.crt
                  path: ca.crt
          - downwardAPI:
              items:
                - path: namespace
                  fieldRef:
                    apiVersion: v1
                    fieldPath: metadata.namespace
        defaultMode: 420
  initContainers:
    - name: hivemq-platform-operator-init
      image: docker.io/hivemq/hivemq-platform-operator-init:1.7.0
      resources:
        limits:
          cpu: 250m
          ephemeral-storage: 1Gi
          memory: 100Mi
        requests:
          cpu: 250m
          ephemeral-storage: 1Gi
          memory: 100Mi
      volumeMounts:
        - name: operator-init
          mountPath: /hivemq
        - name: kube-api-access-h9cqt
          readOnly: true
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      terminationMessagePath: /dev/termination-log
      terminationMessagePolicy: File
      imagePullPolicy: IfNotPresent
  containers:
    - name: hivemq
      image: >-
        registry.hmqc.dev/hivemq-cloud/broker:k8s-4.36.0-monthly-20250205162547-e2bba64
      command:
        - /opt/hivemq/operator/bin/run.sh
      ports:
        - name: mqtts-probe
          containerPort: 1337
          protocol: TCP
        - name: mqtt
          containerPort: 8883
          protocol: TCP
        - name: mqtts-1
          containerPort: 8884
          protocol: TCP
        - name: ws-0
          containerPort: 5883
          protocol: TCP
        - name: ws-1
          containerPort: 5884
          protocol: TCP
        - name: cc
          containerPort: 8080
          protocol: TCP
        - name: api
          containerPort: 8081
          protocol: TCP
        - name: health
          containerPort: 9090
          protocol: TCP
        - name: metrics
          containerPort: 9399
          protocol: TCP
      env:
        - name: JAVA_OPTS
          value: >-
            -XX:+UnlockExperimentalVMOptions -XX:InitialRAMPercentage=50
            -XX:MaxRAMPercentage=50
        - name: HIVEID
          value: 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
        - name: ESE_DATABASE_NAME
          valueFrom:
            secretKeyRef:
              name: pguser-hivemq
              key: dbname
        - name: ESE_DATABASE_PASSWORD
          valueFrom:
            secretKeyRef:
              name: pguser-hivemq
              key: password
        - name: ESE_DATABASE_USER
          valueFrom:
            secretKeyRef:
              name: pguser-hivemq
              key: username
        - name: HIVEMQ_CONFIG_FOLDER
          value: /opt/hivemq/conf-k8s
        - name: HIVEMQ_INTERNAL_ANALYTIC_METRICS
          value: 'true'
        - name: HIVEMQ_INTERNAL_NORMALIZED_MESSAGE_SIZE_IN_BYTES
          value: '5120'
        - name: HIVEMQ_LOGBACK_CONFIG_FOLDER
          value: /opt/hivemq/conf-k8s
        - name: HMQC_METERING_NORMALIZED_MESSAGE_BYTES
          value: '5120'
        - name: HMQC_METERING_PROBE_TOPIC_PREFIX
          value: probes
        - name: OAUTH_SECRET_KEY
          valueFrom:
            secretKeyRef:
              name: oauth-secrets
              key: secret-key
        - name: HIVEMQ_BIND_ADDRESS
          valueFrom:
            fieldRef:
              apiVersion: v1
              fieldPath: status.podIP
      resources:
        limits:
          cpu: '4'
          memory: 3072M
        requests:
          cpu: 400m
          memory: 2048M
      volumeMounts:
        - name: data
          mountPath: /opt/hivemq/data
        - name: logs
          mountPath: /opt/hivemq/log
        - name: licenses
          mountPath: /opt/hivemq/license
        - name: broker-cluster-transport-tls
          readOnly: true
          mountPath: /opt/hivemq/conf/cluster-transport-keystore
        - name: hive-certificates
          readOnly: true
          mountPath: /opt/hivemq/conf/tls
        - name: broker-configuration
          mountPath: /opt/hivemq/conf-k8s/
        - name: pod-info
          mountPath: /etc/podinfo/
        - name: extension-configuration-hivemq-enterprise-security-extension
          mountPath: /opt/hivemq/extensions/hivemq-enterprise-security-extension/conf/
        - name: operator-init
          mountPath: /opt/hivemq/operator/
        - name: kube-api-access-h9cqt
          readOnly: true
          mountPath: /var/run/secrets/kubernetes.io/serviceaccount
      livenessProbe:
        httpGet:
          path: /liveness
          port: 7979
          scheme: HTTP
        initialDelaySeconds: 15
        timeoutSeconds: 1
        periodSeconds: 30
        successThreshold: 1
        failureThreshold: 240
      readinessProbe:
        httpGet:
          path: /readiness
          port: 7979
          scheme: HTTP
        initialDelaySeconds: 3
        timeoutSeconds: 1
        periodSeconds: 5
        successThreshold: 1
        failureThreshold: 3
      terminationMessagePath: /dev/termination-log
      terminationMessagePolicy: File
      imagePullPolicy: IfNotPresent
  restartPolicy: Always
  terminationGracePeriodSeconds: 3600
  dnsPolicy: ClusterFirst
  serviceAccountName: hivemq-platform-pod-broker
  serviceAccount: hivemq-platform-pod-broker
  nodeName: ip-10-255-7-232.eu-central-1.compute.internal
  securityContext:
    fsGroup: 10000
    fsGroupChangePolicy: OnRootMismatch
  imagePullSecrets:
    - name: harbor-pull-secret
  hostname: broker-0
  subdomain: hivemq-broker-cluster
  affinity:
    nodeAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        nodeSelectorTerms:
          - matchExpressions:
              - key: designation
                operator: In
                values:
                  - tier1
    podAntiAffinity:
      requiredDuringSchedulingIgnoredDuringExecution:
        - labelSelector:
            matchExpressions:
              - key: app.kubernetes.io/instance
                operator: In
                values:
                  - 0713bd6e-b9e7-40b0-a1cd-e2ce04c87ec8
          topologyKey: kubernetes.io/hostname
      preferredDuringSchedulingIgnoredDuringExecution:
        - weight: 100
          podAffinityTerm:
            labelSelector:
              matchExpressions:
                - key: app.kubernetes.io/name
                  operator: In
                  values:
                    - hivemq-broker
            topologyKey: topology.kubernetes.io/zone
  schedulerName: default-scheduler
  tolerations:
    - key: designation
      operator: Equal
      value: tier1
      effect: NoSchedule
    - key: node.kubernetes.io/not-ready
      operator: Exists
      effect: NoExecute
      tolerationSeconds: 300
    - key: node.kubernetes.io/unreachable
      operator: Exists
      effect: NoExecute
      tolerationSeconds: 300
  priority: 0
  enableServiceLinks: true
  preemptionPolicy: PreemptLowerPriority
```

# example result of the healthApi
```json
{"status":"UP","components":{"cluster":{"status":"UP","details":{"cluster-id":"2FVes","cluster-nodes":["dZIGZ","fuD1n"],"cluster-size":2,"is-leave-replication-in-progress":false,"node-id":"fuD1n","node-state":"RUNNING"}},"control-center":{"status":"UP","details":{"default-login-mechanism-enabled":true,"enabled":true,"max-session-idle-time":14400},"components":{"control-center-http-listener-8080":{"status":"UP","details":{"bind-address":"0.0.0.0","is-connector-failed":false,"is-connector-open":true,"is-connector-running":true,"port":8080}}}},"extensions":{"status":"UP","components":{"hivemq-cloud-metering-extension":{"status":"UP","details":{"author":"HiveMQ","enabled":true,"name":"HiveMQ Cloud Metering Extension","priority":1000,"start-priority":1000,"startedAt":1750147103100,"version":"03e13fb"},"components":{"internals":{"status":"UP","components":{"entrypoint":{"status":"UP","details":{"started-at":1750147103100}},"license":{"status":"UP","details":{"is-enterprise":true,"is-trial":false,"is-trial-expired":false}},"services":{"status":"UP"}}}}},"hivemq-dns-cluster-discovery":{"status":"UP","details":{"author":"HiveMQ","enabled":true,"name":"DNS Cluster Discovery Extension","priority":1000,"start-priority":10000,"startedAt":1750147102556,"version":"4.3.2"},"components":{"internals":{"status":"UP","components":{"entrypoint":{"status":"UP","details":{"started-at":1750147102556}},"license":{"status":"UP","details":{"is-enterprise":false}},"services":{"status":"UP"}}}}},"hivemq-enterprise-security-extension":{"status":"UP","details":{"author":"HiveMQ","enabled":true,"name":"HiveMQ Enterprise Security Extension","priority":1000,"start-priority":1000,"startedAt":1750147102700,"version":"4.36.0"},"components":{"application":{"status":"UP","components":{"configuration":{"status":"UP","details":{"realms-count":3}}}},"internals":{"status":"UP","components":{"entrypoint":{"status":"UP","details":{"started-at":1750147102700}},"license":{"status":"UP","details":{"is-enterprise":true,"is-trial":false,"is-trial-expired":false}},"services":{"status":"UP"}}}}},"hivemq-prometheus-extension":{"status":"UP","details":{"author":"HiveMQ","enabled":true,"name":"Prometheus Monitoring Extension","priority":1000,"start-priority":1000,"startedAt":1750147102602,"version":"4.0.12"},"components":{"internals":{"status":"UP","components":{"entrypoint":{"status":"UP","details":{"started-at":1750147102602}},"license":{"status":"UP","details":{"is-enterprise":false}},"services":{"status":"UP"}}}}}}},"info":{"status":"UP","details":{"cpu-count":4,"log-level":"INFO","started-at":1750147095405,"version":"4.36.0"}},"liveness-state":{"status":"UP"},"mqtt":{"status":"UP","components":{"mqtts-0":{"status":"UP","details":{"bind-address":"0.0.0.0","is-proxy-protocol-supported":true,"is-running":true,"port":8883,"type":"TCP Listener with TLS"}},"mqtts-1":{"status":"UP","details":{"bind-address":"0.0.0.0","is-proxy-protocol-supported":true,"is-running":true,"port":8884,"type":"TCP Listener with TLS"}},"mqtts-probe":{"status":"UP","details":{"bind-address":"0.0.0.0","is-proxy-protocol-supported":true,"is-running":true,"port":1337,"type":"TCP Listener with TLS"}},"ws-0":{"status":"UP","details":{"allow-extensions":true,"bind-address":"0.0.0.0","is-proxy-protocol-supported":true,"is-running":true,"path":"/mqtt","port":5883,"sub-protocols":["mqttv3.1","mqtt"],"type":"Websocket Listener with TLS"}},"ws-1":{"status":"UP","details":{"allow-extensions":true,"bind-address":"0.0.0.0","is-proxy-protocol-supported":true,"is-running":true,"path":"/mqtt","port":5884,"sub-protocols":["mqttv3.1","mqtt"],"type":"Websocket Listener with TLS"}}}},"readiness-state":{"status":"UP"},"rest-api":{"status":"UP","details":{"authentication-enabled":false,"enabled":true},"components":{"http-listener-8081":{"status":"UP","details":{"bind-address":"0.0.0.0","is-connector-failed":false,"is-connector-open":true,"is-connector-running":true,"port":8081}}}}},"groups":["liveness","readiness"]}
```

# additional information
currently every hivemq pod needs to be queried individually to get the health status, as the healthApi. 
