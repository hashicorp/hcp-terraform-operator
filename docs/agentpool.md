# `AgentPool`

`AgentPool` controller allows managing [Terraform Cloud Agent Pools](https://developer.hashicorp.com/terraform/cloud-docs/agents/agent-pools) and [Terraform Cloud Agent Tokens](https://developer.hashicorp.com/terraform/cloud-docs/users-teams-organizations/api-tokens#agent-api-tokens) via Kubernetes Custom Resources. The Kubernetes CR acts as a single source of truth. It means that all Agent Pool changes made outside of the CR will be returned to the state specified in the CR.

> **Note**
> The controller does not manage the lifecycle of the Terraform Cloud Agents.

## Agent Pool Custom Resorce

Please refer to the [CRD](../config/crd/bases/app.terraform.io_agentpools.yaml) and [API Reference](./api-reference.md#agentpool) to get the full list of available options.

In the following example, we are going to create a new Terraform Cloud Agent Pool with 3 agent tokens. Take a look at the [Prerequisites](./usage.md#prerequisites) before proceeding further.

1. Create a YAML manifest.

    ```yaml
    apiVersion: app.terraform.io/v1alpha2
    kind: AgentPool
    metadata:
      name: this
      namespace: default
    spec:
      organization: kubernetes-operator
      token:
        secretKeyRef:
          name: tfc-operator
          key: token
      name: agent-pool-demo
      agentTokens:
        - name: white
        - name: blue
        - name: red
    ```

2. Apply YAML manifest.

    ```console
    $ kubectl apply -f agentpool.yaml
    ```

3. Wait till the Operator creates a new agent pool `agent-pool-demo` under the `kubernetes-operator` organization and 3 agent tokens: `white`, `blue`, and `red`. You can validate that either by logging in to the Terraform Cloud WEB UI and navigating to the Agent Pools or via CLI.

    Here is an example of the Status and Events outputs of the successfully created Agent Pool and Agent Tokens:

    ```console
    $ kubects describe agentpool this
    ...
    Status:
      Agent Pool ID:  apool-mVnndtTUzdgUsRR3
      Agent Tokens:
        Created At:         1672916079
        Id:                 at-hJmKvjSQQC41aqHn
        Last Used At:       -62135596800
        Name:               red
        Created At:         1672916080
        Id:                 at-fGsUJR4LijkB3k5p
        Last Used At:       -62135596800
        Name:               white
        Created At:         1672916080
        Id:                 at-uCoX1kWU4p5Nk3xq
        Last Used At:       -62135596800
        Name:               blue
      Observed Generation:  1
    Events:
      Type    Reason                Age   From                 Message
      ----    ------                ----  ----                 -------
      Normal  AddFinalizer          10s   AgentPoolController  Successfully added finalizer agentpool.app.terraform.io/finalizer to the object
      Normal  ReconcileAgentPool    9s    AgentPoolController  Status.AgentPoolID is empty, creating a new agent pool
      Normal  ReconcileAgentPool    9s    AgentPoolController  Successfully created a new agent pool with ID apool-mVnndtTUzdgUsRR3
      Normal  ReconcileAgentTokens  7s    AgentPoolController  Reconcilied agent tokens in agent pool ID apool-mVnndtTUzdgUsRR3
      Normal  ReconcileAgentPool    7s    AgentPoolController  Successfully reconcilied agent pool ID apool-mVnndtTUzdgUsRR3
    ```

    Pay attention to `metadata.generation` and `status.observedGeneration` fields. If values are matching, then reconciliation has been completed successfully and Agent Pool and Agent Tokens were created.

    ```console
    Metadata:
      ...
      Generation:  1
    Status:
      Observed Generation:  1
    ```

4. Generated agent tokens are sensitive and thus will be saved in Kubernetes Secrets. The name of the Kubernetes Secret object will be generated automatically and has the following pattern: `<metadata.name>-agent-pool`. For the above example, the name of Secret will be `this-agent-pool`.

    ```console
    $ kubectl get secret this-agent-pool -o yaml
    apiVersion: v1
    data:
      blue: clhWNHE2eH...82cGt0akNZ
      red: YXpTcXkzUE...NKUkdweTVZ
      white: eDVnbGZwT2...o0SFBacHpZ
    kind: Secret
    metadata:
      creationTimestamp: "2023-01-05T10:54:39Z"
      labels:
        agentPoolID: apool-mVnndtTUzdgUsRR3
      name: this-agent-pool
      namespace: default
      ownerReferences:
      - apiVersion: app.terraform.io/v1alpha2
        blockOwnerDeletion: true
        controller: true
        kind: AgentPool
        name: this
        uid: c157c3d0-6621-4f8d-9a46-f4a22a9e5a9d
      resourceVersion: "34192"
      uid: 5e64ad8c-3ce9-4744-a7e7-5d32f1b0ca3b
    type: Opaque
    ```

5. If you want to add a new agent token, you need to modify the YAML manifest and then apply changes. Wait till the Operator applies changes and then you can find a new token in Kubernetes Secrets:

    ```yaml
    apiVersion: app.terraform.io/v1alpha2
    kind: AgentPool
    metadata:
      name: this
      namespace: default
    spec:
      organization: kubernetes-operator
      token:
        secretKeyRef:
          name: tfc-operator
          key: token
      name: agent-pool-demo
      agentTokens:
        - name: white
        - name: blue
        - name: red
        - name: green
    ```

    ```console
    $ kubectl apply -f agentpool.yaml
    ```

    ```console
    $ kubectl get secret this-agent-pool -o yaml
    apiVersion: v1
    data:
      blue: clhWNHE2eH...82cGt0akNZ
      green: SVZ0dEZ5UF...lyTFYyM29B
      red: YXpTcXkzUE...NKUkdweTVZ
      white: eDVnbGZwT2...o0SFBacHpZ
    kind: Secret
    metadata:
      creationTimestamp: "2023-01-05T10:54:39Z"
      labels:
        agentPoolID: apool-mVnndtTUzdgUsRR3
      name: this-agent-pool
      namespace: default
      ...
    ```

6. If you want to delete an existing agent token, you need to modify the YAML manifest and then apply changes. Wait till the Operator applies changes:

    ```yaml
    apiVersion: app.terraform.io/v1alpha2
    kind: AgentPool
    metadata:
      name: this
      namespace: default
    spec:
      organization: kubernetes-operator
      token:
        secretKeyRef:
          name: tfc-operator
          key: token
      name: agent-pool-demo
      agentTokens:
        - name: blue
    ```

    ```console
    $ kubectl apply -f agentpool.yaml
    ```

    ```console
    $ kubectl get secret this-agent-pool -o yaml
    apiVersion: v1
    data:
      blue: clhWNHE2eH...82cGt0akNZ
    kind: Secret
    metadata:
      creationTimestamp: "2023-01-05T10:54:39Z"
      labels:
        agentPoolID: apool-mVnndtTUzdgUsRR3
      name: this-agent-pool
      namespace: default
      ...
    ```

If you encounter any issues with the `AgentPool` controller please refer to the [Troubleshooting](../README.md#troubleshooting).
