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

## General Questions

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

  Behavior may vary from controller to controller. Most probably you will notice that CR objects are constantly reconciled and that might cause constant updates of Terraform Cloud objects. For example, the `Module` controller might trigger a new run every reconciliation and because of that the Run queue will grow infinitely and consume all resources.
  
  It is definitely better to avoid such situations.

## Workspace Controller

## Module Controller

## Agent Pool Controller

- **What will happen if I delete an Agent Pool CR?**

  The Agent Pool controller will delete Agent Pool from Terraform Cloud, as well as the Kubernetes Secret that stores the Agent Tokens that were generated for this pool.
