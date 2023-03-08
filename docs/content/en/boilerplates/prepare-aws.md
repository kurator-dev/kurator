Download the latest binary of clusterawsadm from the [AWS provider releases](https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/tag/v2.0.0). The clusterawsadm command line utility assists with identity and access management (IAM) for [Cluster API Provider AWS](https://cluster-api-aws.sigs.k8s.io/).

```console
# Download the latest release
curl -L https://github.com/kubernetes-sigs/cluster-api-provider-aws/releases/download/v2.0.0/clusterawsadm-linux-amd64 -o clusterawsadm
# Make it executable
chmod +x clusterawsadm
# Move the binary to a directory present in your PATH
sudo mv clusterawsadm /usr/local/bin
# Check version to confirm installation
clusterawsadm version

export AWS_REGION=us-east-1 # This is used to help encode your environment variables
# Get AK/SK from https://docs.aws.amazon.com/IAM/latest/UserGuide/id_credentials_access-keys.html
export AWS_ACCESS_KEY_ID=<your-access-key>
export AWS_SECRET_ACCESS_KEY=<your-secret-access-key>

# The clusterawsadm utility takes the credentials that you set as environment
# variables and uses them to create a CloudFormation stack in your AWS account
# with the correct IAM resources.
clusterawsadm bootstrap iam create-cloudformation-stack

# Create the base64 encoded credentials using clusterawsadm.
# This command uses your environment variables and encodes
# them in a value to be stored in a Kubernetes Secret.
export AWS_B64ENCODED_CREDENTIALS=$(clusterawsadm bootstrap credentials encode-as-profile)
```
