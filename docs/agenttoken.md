# `AgentToken`

The `AgentToken` controller allows managing tokens in arbitrary agent pools. The controller supports two policies that define its behavior in token management:
  - `merge` — the controller manages its tokens alongside existing tokens in the pool, without modifying or deleting tokens it does not own.
  - `owner` — the controller assumes full ownership of all tokens in the pool, managing and potentially modifying or deleting tokens, including those it did not create.

The `merge` policy can be especially useful in multicluster environments, where the operator runs in different clusters and each instance manages its own tokens.

## Agent Token Custom Resorce

Below is a basic example of an `AgentToken` Custom Resource:

Please refer to the [CRD](../config/crd/bases/app.terraform.io_agenttokens.yaml) and [API Reference](./api-reference.md#agenttoken) to get the full list of available options.

```yaml
apiVersion: app.terraform.io/v1alpha2
kind: AgentToken
metadata:
  name: this
spec:
  organization: kubernetes-operator
  token:
    secretKeyRef:
      name: hcp-terraform-operator
      key: token
  agentPool:
    name: multik
  agentTokens:
  - name: token-a
  - name: token-b
  secretName: this
  deletionPolicy: destroy
  managementPolicy: merge
```

Once the above CR is applied, the Operator will create two tokens, `token-a` and `token-b`, in the agent pool `multik`. It will only manage these tokens (ensuring they exist) without affecting existing ones, because the default `spec.managementPolicy` is set to `merge`.

If you have any questions, please check out the [FAQ](./faq.md#agent-token-controller) to see if you can find answers there.

If you encounter any issues with the `AgentToken` controller please refer to the [Troubleshooting](../README.md#troubleshooting).
