# Overview

E2E testing of the vSphere extension requires a vSphere instance for the testing to run against. Access to a vSphere instance is problematic:
1. Gardener uses [Test Machinery](https://github.com/gardener/test-infra/tree/master/docs/testmachinery) to manage the matrix of tests defined for an artifact. 
2. Artifacts are distributed to the public. To avoid supply chain contamination, they are built in the Gardener "live" landscape. Live is a secure landscape with tight access controls.

For it's part, vSphere is a complex product and it is not advisable to have it directly exposed to the public internet due to various vulnerabilities that emerge over time. That said, it is a benign entity that is not itself a threat. A GCVE instance with default cordons starts in an inaccessible silo that is dependent on
standard GCP network constructs. 

The GCVE APIs require an administrator entitlement to enable them for a GCP project and standard IAM for them to be used by entities.  

After new TM instance outside of live was rejected, two high-level configurations were remained to provide a vSphere endpoint for E2E testing:

## GCVE in Live
In this configuration, setup for a Test Machinery run would create a vSphere instance using GCVE directly in the live landscape. This siloed
vSphere cluster is created, used and destroyed by code that is already trusted by Test Machinery. Since the cluster is treated atomically, no artifacts of a cluster remain 
after a test. 

As the cluster is only populated by the trusted code, there is transitive sterility of the live environment. This is quite a bit simpler in implementation
since all networking is contained on the same GCP subnets as the live Test Machinery. It is also significantly more secure as no additional credentials are required for the execution of the stack and
there is zero exposure to the outside world.

## Testmachinery in Testmachinery
As sometimes happens, the most complex situation with the largest number of workarounds and failure modes was eventually implemented:

1. The build creates a test container that includes TM, cc-config and custom binaries
2. CI executes a script on this container
3. Keys are retrieved and the custom binary creates a GCVE instance. This takes ~2.5 hours if the instance does not exist, but is idempotent and can reuse an existing instance.
4. Terraform creates a new GKE cluster in the same GCP project as the vSphere instance 
5. A second copy of Test Machinery is installed in the GKE cluster and launched, including a new S3 database instance. 
6. The E2E tests are run in this isolated TM cluster
7. The results are for all intents discarded because there is no way to get them back to the original TM instance

### To be continued....