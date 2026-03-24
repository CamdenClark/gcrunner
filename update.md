# Update gcrunner

<walkthrough-tutorial-duration duration="5"></walkthrough-tutorial-duration>

This tutorial walks you through updating gcrunner to the latest version.

<walkthrough-project-setup billing="true"></walkthrough-project-setup>

Click **Next** to begin.

## Project setup

Select your project above, then set the project ID:

```sh
export PROJECT_ID=<walkthrough-project-id/>
gcloud config set project $PROJECT_ID
```

## Check out the latest release

Pull the latest version of gcrunner:

```sh
cd ~
git clone https://github.com/camdenclark/gcrunner
cd gcrunner
```

## Initialize Terraform

Connect to your existing Terraform state bucket:

```sh
cd terraform
terraform init -backend-config="bucket=${PROJECT_ID}-gcrunner-tfstate" -backend-config="prefix=gcrunner"
```

## Apply the update

Set your variables:

```sh
export TF_VAR_project_id=$PROJECT_ID
export TF_VAR_region=us-central1
```

Review the planned changes:

```sh
terraform plan
```

Apply when ready:

```sh
terraform apply
```

Type `yes` when prompted. Terraform will update the Cloud Run service to the latest gcrunner image and apply any infrastructure changes.

## Conclusion

<walkthrough-conclusion-trophy></walkthrough-conclusion-trophy>

gcrunner is now up to date. Check [gcrunner.com](https://gcrunner.com) for the latest documentation.
