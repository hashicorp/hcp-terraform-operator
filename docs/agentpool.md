# `AgentPool`

`AgentPool` controller allows managing Terraform Cloud Agent Pools via Kubernetes Custom Resources. The controller does not manage the lifecycle of the Agents.

Below is an example of a Workspace Custom Resource:

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: AgentPool
metadata:
  name: this
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

Once the above CR is applied, the Operator creates a new agent pool `agent-pool-demo` under the `kubernetes-operator` organization and 3 agent tokens: `white`, `blue`, and `red`.

Generated agent tokens are sensitive and thus will be saved in Kubernetes Secrets. The name of the Kubernetes Secret object will be generated automatically and has the following pattern: `<metadata.name>-agent-pool`. For the above example, the name of Secret will be `this-agent-pool`.

Please refer to the [CRD](../config/crd/bases/app.terraform.io_agentpools.yaml) to get the full list of available options.
