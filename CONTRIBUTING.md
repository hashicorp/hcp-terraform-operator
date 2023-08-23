# Contributor Guide

We appreciate your enthusiasm for participating in the development of the Terraform Cloud Operator. Your contributions are warmly welcomed. Here you will find instructions on how to contribute to the Terraform Cloud Operator.

If you're a newcomer to the realm of Kubernetes Operators and are eager to expand your knowledge, a great place to begin your journey is by exploring the [Kubebuilder Book](https://book.kubebuilder.io/).

## Contribution

### Updating documentation

We appreciate your interest in improving our documentation! Please find our documentation guidelines below:

1. Follow the existing documentation style and structure to maintain consistency.

1. Use clear and concise language that is easy for readers to understand.

1. If you're introducing new concepts, consider providing examples or code snippets to illustrate usage.

1. If your changes require updates to screenshots or diagrams, ensure they are up-to-date and accurately represent the content.

Please submit your changes.

### Requesting new features

We appreciate your enthusiasm for enhancing the Operator with new features! To request a new feature, follow the steps below:

1. **Check Existing Requests**: before submitting a new feature request, search our issue tracker to ensure that your idea hasn't been suggested before. If a similar request already exists, you can contribute to that discussion instead.

1. **Create a New Issue**: if you don't find an existing feature request that matches your idea, create a new issue in our issue tracker on GitHub. Use the "New Issue" template labeled "🚀 Feature request".

1. **Provide Details**: in the issue, clearly describe the new feature you're proposing. Include the problem the feature aims to solve and why it's valuable.

1. **Use a Clear Title**: choose a descriptive and succinct title for your feature request. This helps other contributors and maintainers quickly understand the essence of your suggestion.

1. **Additional Context**: If applicable, provide context such as use cases, examples, or scenarios where the feature would be beneficial.

### Reporting bugs

We greatly appreciate your assistance in improving the Operator by identifying and reporting bugs. To report a bug, follow the steps below:

1. **Check Existing Issues**: before submitting a new bug report, search our issue tracker to see if the bug has already been reported. If you find a similar issue, you can contribute to that discussion with additional information.

1. **Create a New Issue**: if you don't find an existing bug report that matches the issue you've encountered, create a new issue in our issue tracker on GitHub. Use the "New Issue" template labeled "🐛 Bug report".

1. **Provide Details**: in the issue, provide clear and concise details about the bug you've encountered. Include information such as steps to reproduce the bug, the expected behavior, and the actual behavior you observed.

1. **Use a Descriptive Title**: choose a descriptive title for your bug report. This helps other contributors and maintainers quickly understand the nature of the issue.

1. **Additional Context**: if applicable, include any relevant information such as screenshots, error messages, or logs that might aid in diagnosing the issue.

### Proposing bug fixes

If you've identified a bug in the Operator and have written a patch to fix it, we greatly appreciate your contribution to improving the Operator's stability. Please follow the steps below in this document to set up a development environment, tests your changes and submit them.

### Cosmetic changes

We appreciate your attention to detail in ensuring the quality of the Operator's codebase, even in cosmetic and formatting aspects. While these changes might not directly address functional issues, they contribute to code readability and maintainability. Please submit your changes.

### Adding new features or changing existing features

We value your desire to contribute by introducing new features or enhancing existing ones in the Operator. Your contributions play a pivotal role in the evolution and growth of the Operator. Please create a new issue in our issue tracker on GitHub to have a discussion with the maintainers and involve the community before submitting a code change. Use the "New Issue" template labeled "🚀 Feature request".

## Development Environment

1. Create an API token that you are going to use for development

    We strongly advise creating a separate account, organization, and API token for development purposes. A free [Terraform Cloud](https://app.terraform.io/) account is more than enough for that purpose. Follow the steps in the [Usage guide](./docs/usage.md#prerequisites) to get more information on how to generate a token and keep it in the Kubernetes Secret.

1. Install Go

    Install the version of [Golang](https://golang.org/) as indicated in the [`go.mod`](./go.mod) file.

1. Clone this repo

    ```console
    $ git clone https://github.com/hashicorp/terraform-cloud-operator.git
    $ cd terraform-cloud-operator
    ```

1. Prepare a Kubernetes cluster

    While our preference is to use [kind](https://kind.sigs.k8s.io/) for setting up a Kubernetes cluster for development and test purposes, feel free to opt for the solution that best suits your preferences.

1. Install CRDs into the Kubernetes cluster

    ```console
    $ make install
    ```

1. Run the Operator on your cluster

    ```console
    $ make run
    ```

    In this scenario, all outputs will be displayed in the console. You should be able to see an output similar to this:

    ```console
    INFO	Starting EventSource	{"controller": "workspace", "controllerGroup": "app.terraform.io", "controllerKind": "Workspace", "source": "kind source: *v1alpha2.Workspace"}
    INFO	Starting Controller	{"controller": "workspace", "controllerGroup": "app.terraform.io", "controllerKind": "Workspace"}
    INFO	Starting EventSource	{"controller": "agentpool", "controllerGroup": "app.terraform.io", "controllerKind": "AgentPool", "source": "kind source: *v1alpha2.AgentPool"}
    INFO	Starting Controller	{"controller": "agentpool", "controllerGroup": "app.terraform.io", "controllerKind": "AgentPool"}
    INFO	Starting EventSource	{"controller": "module", "controllerGroup": "app.terraform.io", "controllerKind": "Module", "source": "kind source: *v1alpha2.Module"}
    INFO	Starting Controller	{"controller": "module", "controllerGroup": "app.terraform.io", "controllerKind": "Module"}
    INFO	Starting workers	{"controller": "agentpool", "controllerGroup": "app.terraform.io", "controllerKind": "AgentPool", "worker count": 1}
    INFO	Starting workers	{"controller": "workspace", "controllerGroup": "app.terraform.io", "controllerKind": "Workspace", "worker count": 1}
    INFO	Starting workers	{"controller": "module", "controllerGroup": "app.terraform.io", "controllerKind": "Module", "worker count": 1}
    ```

Congratulations! You're all set to start working on your change!

## Make changes

### Apply changes

There are two main pieces of the Operator `API` and `Controller`. Depending on which part you are working on, you need to perform pass different sets of targets to the `make` command:

- API:

    Once you made changes in the API codebase(`/api/**`), you need to re-generate CRDs and re-run the Operator to apply them. Press `Ctrl+C` in the console where you have it running, generate new CRDs, install them and run again. It is all can be done in one command:

    ```console
    $ make generate manifests install run
    ```

- Controller:

    Once you made changes in the controllers codebase(`/controllers/**`), you need to re-run the Operator to apply them. Press `Ctrl+C` in the console where you have it running and run again:

    ```console
    $ make run
    ```
    
    For simplicity, you can always run the full set of targets to re-run the Operator regardless of the changes you have made:

    ```console
    $ make generate manifests install run
    ```

### Test changes

- API:

    If your made API-related changes(`/api/**`), please make sure you have updated the tests and always run them:

    ```console
    $ make test-api
    ```
    
- Controllers:

    We rely on (Ginkgo)[https://github.com/onsi/ginkgo] testing framework in our controllers E2E tests. If your made controller-related changes(/controllers/**), please make sure you have updated the tests and always run them.

    Export the organization name and API token to environment variable:

    ```console
    export TFC_TOKEN=<YOUR_API_TOKEN>
    export TFC_ORG=<YOUR_ORG_NAME>
    ```

    Run tests:

    ```console
    $ make test
    ```
    
    In all other cases, please follow the rules:

    - Always run tests before and after your changes.

    - Write tests to cover your changes.

    Every test should be executable through a make target, and the target should have a prefix of "test-".

## Submitting Changes

### Creating a Pull Request

We're excited that you're ready to contribute to the Operator by creating a pull request (PR)! Pull requests are a fundamental way to propose and discuss changes with the project maintainers and contributors.

1. **Description**: write a detailed description of your changes. Keep in mind, that it should be clear why you make this change, what you have changed, and how this will affect the Operator users. If you are working on a fix for an existing issue, you can provide less details about it.

1. **Usage Example**: if your change is API-related, make sure you have provided **Before** and **After** usage examples.

1. **Release Note**: write down a change log within the PR and update the [CHANGELOG.MD](./CHANGELOG.md) file respectively.

1. **References**: if you fix an existing issue, please provide a reference to it by using a relevant [GitHub keyword](https://docs.github.com/en/get-started/writing-on-github/working-with-advanced-formatting/using-keywords-in-issues-and-pull-requests#linking-a-pull-request-to-an-issue).

### Review Process

1. Your pull request will be reviewed by maintainers.

1. Address any feedback or suggestions provided by reviewers.

1. Once the pull request is approved, it will be merged into the main branch of the project repository.
