# EXAMPLE: Standalone Mounted Installation of Terraform Enterprise

## About This Example

This example for Terraform Enterprise creates a TFE installation with the following traits:

- Standalone
- Mounted Disk production type
- n1-standard-4 virtual machine type
- Ubuntu 20.04
- A publicly accessible HTTP load balancer with TLS termination


## Prerequisites

This example assumes that the following resources exist:

- TFE license is on a file path defined by `var.license_file` 
- A DNS zone
- Valid managed SSL certificate to use with load balancer:
  - Create/Import a managed SSL Certificate in Network Services -> Load Balancing to serve as the certificate for the DNS A Record.
  
## How to Use This Module

### Deployment

1. Read the entire [README.md](../../README.md) of the root module.
2. Ensure account meets module prerequisites from above.
3. Clone repository.
4. Change directory into desired example folder.
5. Create a local `terraform.auto.tfvars` file and instantiate the required inputs as in the respective `./examples/standalone-mounted/variables.tf` including the path to the license under the `license_file` variable value.
6. Authenticate against the Google provider. See [instructions](https://registry.terraform.io/providers/hashicorp/google/latest/docs/guides/provider_reference#authentication).
7. Initialize terraform and apply the module configurations using the commands below:

  NOTE: `terraform plan` will print out the execution plan which describes the actions Terraform will take in order to build your infrastructure to match the module configuration. If anything in the plan seems incorrect or dangerous, it is safe to abort here and not proceed to `terraform apply`.

  ```
  terraform init
  terraform plan
  terraform apply
  ```

## Post-deployment Tasks

The build should take approximately 10-15 minutes to deploy. Once the module has completed, give the platform another 10 minutes or so prior to attempting to interact with it in order for all containers to start up.

Unless amended, this example will not create an initial admin user using the IACT, but it does output the URL for your convenience. Follow the advice in this document to create the initial admin user, and log into the system using this user in order to configure it for use.

### Connecting to the TFE Console

The TFE Console is only available in a standalone environment

1. Navigate to the URL supplied via `tfe_console_url` Terraform output
2. Copy the `tfe_console_password` Terraform output
3. Enter the console password
4. Click `Unlock`

### Connecting to the TFE Application

1. Navigate to the URL supplied via the `login_url` Terraform output. (It may take several minutes for this to be available after initial deployment. You may monitor the progress of cloud init if desired on one of the instances.)
2. Enter a `username`, `email`, and `password` for the initial user.
3. Click `Create an account`.
4. After the initial user is created you may access the TFE Application normally using the URL supplied via `login_url` Terraform output.