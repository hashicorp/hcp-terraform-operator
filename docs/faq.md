# Frequently Asked Questions

## :warning: Beta Version Questions :warning:

- **I am getting a '403' error when trying to install Terraform Cloud Operator v2 beta. How to address that?**

  Make sure you are logged out from "public.ecr.aws":

  ```console
  $ docker logout public.ecr.aws
  ```

## Terminology

- **What is a Kubernetes Operator?**

  Operators are software extensions to Kubernetes that make use of custom resources to manage applications and their components. More information is in the [Kubernetes Documentation](https://kubernetes.io/docs/concepts/extend-kubernetes/operator/).

- **What is a Kubernetes Controller?**

  In Kubernetes, controllers are control loops that watch the state of your cluster, then make or request changes where needed. Each controller tries to move the current cluster state closer to the desired state. More information is in the [Kubernetes Documentation](https://kubernetes.io/docs/concepts/architecture/controller/).

- **What is a Kubernetes Custom Resource?**

  Custom resources are extensions of the Kubernetes API. More information is in the [Kubernetes Documentation](https://kubernetes.io/docs/concepts/extend-kubernetes/api-extension/custom-resources/).

## General Questions

- **What is the difference between versions `v1` and `v2` of the Operator?**

  The second version of the Operator was developed to address some major concerns that we encountered in the first version.

  Here is the list of major improvements in the version 2:

  - A new operator option `--namespace` allows configuration of namespaces to watch. It can be one of the following: all, single, or multiple namespaces. By default, the Operator watches all namespaces, and as your setup grows, you can have multiple deployments of the Operator to better handle the load.

  - A new operator option `--sync-period` allows configuration of the minimum frequency at which all watched resources are reconciled. This allows faster synchronization of the state between Custom Resources and Terraform Cloud.

  - The Operator manages a Terraform Cloud client for each Custom Resource. This means that a single deployment of the Operator can work across multiple Terraform Cloud organizations.

  - The Operator consists of multiple controllers that manage different Terraform Cloud resources. This provides additional flexibility, e.g. a module can be executed in a workspace that is not managed by the Operator. More details about controllers you can find in the [README](../README.md) file.

  - Each controller has the option to manage the number of workers it has. By default, each controller has 1 worker. A worker is a thread that runs the control loop for a given Custom Resource. The more workers the controller has, the more Customer Resources it can handle concurrently. This improves the Operator's performance. Please refer to the [performance FAQ section](./faq.md#performance) to better understand the pros and cons.

  - Additional technical improvements:

    - More detailed logging
    - Controllers produce event messages for each Custom Resource

    - Better coverage of features supported by Terraform Cloud, more information [here](./features.md);

    - Better test coverage 

    - A leaner "Distroless" container image for deployment that is built for more platforms. More information [here](https://github.com/GoogleContainerTools/distroless).


- **Can a single deployment of the Operator watch single, multiple, or all namespaces?**

  Yes, a single deployment of the Operator can either watch a single namespace, multiple namespaces, or all namespaces in the Kubernetes cluster. By default, the Operator watches all namespaces. If you want to specify single or multiple namespaces, you need to pass the following option when installing or upgrading the Helm chart.

  *watch a single namespace*
  ```console
  $ helm ... --set 'operator.watchedNamespaces={red}'
  ```

  *watch multiple namespaces*
  ```console
  $ helm ... --set 'operator.watchedNamespaces={white,blue,red}'
  ```

- **What will happen if I have multiple deployments of the Operator watching the same namespace(s)?**

Unexpected behaviour is likely when multiple deployments of the operator try to reconcile the same resource. Most likely you will notice that Customer Resource objects are constantly reconciled and this can cause constant updates of Terraform Cloud objects. For example, the `Module` controller might trigger a new run every reconciliation and because of that the Run queue could grow infinitely.

  It is definitely better to avoid such situations.

- **What do the `*-workers` options do?**

  The `*-workers` options allow configuration of the number of concurrent workers available to process changes to resources. In certain cases increasing this number can improve performance.

- **What does the `sync-period` option do?**

  The `--sync-period` option specifies the minimum frequency at which watched resources are reconciled. The synchronization period should be aligned with the number of managed Customer Resources. If the period is too low and the number of managed resources is too high, you may observe slowness in synchronization.

## Performance

- **How many Custom Resources can be managed by a single deployment of the Operator?**

  In theory, a single deployment of the Operator can manage thousands of resources. However, the Operator's performance depends on the number of API calls it does and the Terraform Cloud API [rate limit](https://developer.hashicorp.com/terraform/cloud-docs/api-docs#rate-limiting) for the token used.

  The number of API calls the Operator does depends on multiple factors:

    - The value of the `sync-period` option;

    - The values of `*-workers` options.

    - The type of the resource.

    - The Terraform Cloud features being used.

  With the default values of `sync-period` (5 minutes) and `*-workers` (1 worker per controller), we recommend managing **100 resources per token**. This number can vary based on previously mentioned factors. This number can be updated later to accommodate  changes in the Terraform Cloud API.

## Workspace Controller

- **Can a single deployment of the Operator manage the Workspaces of different Organizations?**

  Yes, it can. Workspace Customer Resource has mandatory fields `spec.organization` and `spec.token`. The Operator manages workspaces based on these credentials.

- **Where can I find Workspace outputs?**

  Non-sensitive outputs will be saved in Kubernetes ConfigMap. Sensitive outputs will be saved in Kubernetes Secret. In both cases, the name of the corresponding Kubernetes object will be generated automatically and has the following pattern: `<CR.metadata.name>-outputs`.

## Module Controller

- **Where can I find Module outputs?**

  Non-sensitive outputs will be saved in a ConfigMap. Sensitive outputs will be saved in a Secret. In both cases, the name of the corresponding Kubernetes object will be generated automatically and has the following pattern: `<metadata.name>-module-outputs`. When the underlying workspace is managed by the operator, all outputs will be duplicated in the corresponding ConfigMap or Secret.

- **Can I execute a new Run without changing any Workspace or Module attributes?**

Yes. There is a special attribute `spec.restartedAt` that you need to update in order to trigger a new Run execution. For example:

	```console
  $ kubectl patch module <NAME> --type=merge --patch '{"spec": {"restartedAt": "'`date -u -Iseconds`'"}}'
  ```

## Agent Pool Controller

- **Where can I find Agent tokens?**

  The Agent tokens are sensitive and will be saved in a Secret. The name of the Secret object will be generated automatically and has the following pattern: `<metadata.name>-agent-pool`.

- **Does the Operator restore tokens if I delete the whole Secret containing the Agent Tokens or a single token from it?**

  No. You will have to update the Custom Resource to re-create tokens.

- **What will happen if I delete an Agent Pool Customer Resource?**

  The Agent Pool controller will delete Agent Pool from Terraform Cloud, as well as the Kubernetes Secret that stores the Agent Tokens that were generated for this pool.
