<p>Packages:</p>
<ul>
<li>
<a href="#vsphere.provider.extensions.gardener.cloud%2fv1alpha1">vsphere.provider.extensions.gardener.cloud/v1alpha1</a>
</li>
</ul>
<h2 id="vsphere.provider.extensions.gardener.cloud/v1alpha1">vsphere.provider.extensions.gardener.cloud/v1alpha1</h2>
<p>
<p>Package v1alpha1 contains the vSphere provider API resources.</p>
</p>
Resource Types:
<ul><li>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>
</li><li>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig</a>
</li><li>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureConfig">InfrastructureConfig</a>
</li><li>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.WorkerStatus">WorkerStatus</a>
</li></ul>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig
</h3>
<p>
<p>CloudProfileConfig contains provider-specific configuration that is embedded into Gardener&rsquo;s <code>CloudProfile</code>
resource.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
vsphere.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>CloudProfileConfig</code></td>
</tr>
<tr>
<td>
<code>namePrefix</code></br>
<em>
string
</em>
</td>
<td>
<p>NamePrefix is used for naming NSX-T resources</p>
</td>
</tr>
<tr>
<td>
<code>folder</code></br>
<em>
string
</em>
</td>
<td>
<p>Folder is the vSphere folder name to store the cloned machine VM (worker nodes)</p>
</td>
</tr>
<tr>
<td>
<code>regions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.RegionSpec">
[]RegionSpec
</a>
</em>
</td>
<td>
<p>Regions is the specification of regions and zones topology</p>
</td>
</tr>
<tr>
<td>
<code>defaultClassStoragePolicyName</code></br>
<em>
string
</em>
</td>
<td>
<p>DefaultClassStoragePolicyName is the name of the vSphere storage policy to use for the &lsquo;default&rsquo; StorageClass.</p>
</td>
</tr>
<tr>
<td>
<code>failureDomainLabels</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.FailureDomainLabels">
FailureDomainLabels
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>FailureDomainLabels are the tag categories used for regions and zones.</p>
</td>
</tr>
<tr>
<td>
<code>dnsServers</code></br>
<em>
[]string
</em>
</td>
<td>
<p>DNSServers is a list of IPs of DNS servers used while creating subnets.</p>
</td>
</tr>
<tr>
<td>
<code>dhcpOptions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.DHCPOption">
[]DHCPOption
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DHCPOptions contains optional options for DHCP like Domain name, NTP server,&hellip;</p>
</td>
</tr>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImages">
[]MachineImages
</a>
</em>
</td>
<td>
<p>MachineImages is the list of machine images that are understood by the controller. It maps
logical names and versions to provider-specific identifiers.</p>
</td>
</tr>
<tr>
<td>
<code>constraints</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Constraints">
Constraints
</a>
</em>
</td>
<td>
<p>Constraints is an object containing constraints for certain values in the control plane config.</p>
</td>
</tr>
<tr>
<td>
<code>csiResizerDisabled</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>CSIResizerDisabled is a flag to disable the CSI resizer (e.g. resizer is not supported for vSphere 6.7)</p>
</td>
</tr>
<tr>
<td>
<code>machineTypeOptions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineTypeOptions">
[]MachineTypeOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineTypeOptions is the list of machine type options to set additional options for individual machine types.</p>
</td>
</tr>
<tr>
<td>
<code>dockerDaemonOptions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.DockerDaemonOptions">
DockerDaemonOptions
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DockerDaemonOptions contains configuration options for docker daemon service</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig
</h3>
<p>
<p>ControlPlaneConfig contains configuration settings for the control plane.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
vsphere.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>ControlPlaneConfig</code></td>
</tr>
<tr>
<td>
<code>cloudControllerManager</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudControllerManagerConfig">
CloudControllerManagerConfig
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>CloudControllerManager contains configuration settings for the cloud-controller-manager.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerClasses</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CPLoadBalancerClass">
[]CPLoadBalancerClass
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>LoadBalancerClasses lists the load balancer classes to be used.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerSize</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>LoadBalancerSize can override the default of the NSX-T load balancer size (&ldquo;SMALL&rdquo;, &ldquo;MEDIUM&rdquo;, or &ldquo;LARGE&rdquo;) defined in the cloud profile.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureConfig">InfrastructureConfig
</h3>
<p>
<p>InfrastructureConfig infrastructure configuration resource</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
vsphere.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>InfrastructureConfig</code></td>
</tr>
<tr>
<td>
<code>networks</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Networks">
Networks
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>Networks contains optional existing network infrastructure to use.
If not defined, NSX-T Tier-1 gateway and load balancer are created for the shoot cluster.</p>
</td>
</tr>
<tr>
<td>
<code>overwriteNSXTInfraVersion</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>OverwriteNSXTInfraVersion allows to fix the ensurer version used to create the NSXT-T infrastructure.
This is an advanced configuration to overwrite the automatic version selection.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.WorkerStatus">WorkerStatus
</h3>
<p>
<p>WorkerStatus contains information about created worker resources.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>apiVersion</code></br>
string</td>
<td>
<code>
vsphere.provider.extensions.gardener.cloud/v1alpha1
</code>
</td>
</tr>
<tr>
<td>
<code>kind</code></br>
string
</td>
<td><code>WorkerStatus</code></td>
</tr>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImage">
[]MachineImage
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is a list of machine images that have been used in this worker. Usually, the extension controller
gets the mapping from name/version to the provider-specific machine image data in its componentconfig. However, if
a version that is still in use gets removed from this componentconfig it cannot reconcile anymore existing <code>Worker</code>
resources that are still using this version. Hence, it stores the used versions in the provider status to ensure
reconciliation is possible.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.AdvancedDHCPState">AdvancedDHCPState
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.NSXTInfraState">NSXTInfraState</a>)
</p>
<p>
<p>AdvancedDHCPState holds IDs of objects managed with the NSX-T Advanced API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>logicalSwitchID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>profileID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>serverID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>portID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipPoolID</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.CPLoadBalancerClass">CPLoadBalancerClass
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig</a>)
</p>
<p>
<p>CPLoadBalancerClass provides the name of a load balancer</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>ipPoolName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPPoolName is the name of the NSX-T IP pool.</p>
</td>
</tr>
<tr>
<td>
<code>tcpAppProfileName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TCPAppProfileName is the profile name of the load balaner profile for TCP</p>
</td>
</tr>
<tr>
<td>
<code>udpAppProfileName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>UDPAppProfileName is the profile name of the load balaner profile for UDP</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudControllerManagerConfig">CloudControllerManagerConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.ControlPlaneConfig">ControlPlaneConfig</a>)
</p>
<p>
<p>CloudControllerManagerConfig contains configuration settings for the cloud-controller-manager.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>featureGates</code></br>
<em>
map[string]bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>FeatureGates contains information about enabled feature gates.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.Constraints">Constraints
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>Constraints is an object containing constraints for the shoots.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>loadBalancerConfig</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.LoadBalancerConfig">
LoadBalancerConfig
</a>
</em>
</td>
<td>
<p>LoadBalancerConfig contains constraints regarding allowed values of the &lsquo;Lo&rsquo; block in the control plane config.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.DHCPOption">DHCPOption
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>, 
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.RegionSpec">RegionSpec</a>)
</p>
<p>
<p>DHCPOption contains a DHCP option by code</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>code</code></br>
<em>
int
</em>
</td>
<td>
<p>Code is the tag according to the BOOTP Vendor Extensions and DHCP Options (see <a href="https://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml">https://www.iana.org/assignments/bootp-dhcp-parameters/bootp-dhcp-parameters.xhtml</a>)
most important codes: &lsquo;Domain Name&rsquo;=15 (only allowed for NSX-T 2.5, use code 119 for NSX-T &gt;= 3.0), &lsquo;NTP server&rsquo;=42, &lsquo;Domain Search&rsquo;: 119</p>
</td>
</tr>
<tr>
<td>
<code>values</code></br>
<em>
[]string
</em>
</td>
<td>
<p>Values are the values for the given code</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.DockerDaemonOptions">DockerDaemonOptions
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>DockerDaemonOptions contains configuration options for Docker daemon service</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>httpProxyConf</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HTTPProxyConf contains HTTP/HTTPS proxy configuration for Docker daemon</p>
</td>
</tr>
<tr>
<td>
<code>insecureRegistries</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>InsecureRegistries adds the given registries to Docker on the worker nodes
(see <a href="https://docs.docker.com/registry/insecure/">https://docs.docker.com/registry/insecure/</a>)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.FailureDomainLabels">FailureDomainLabels
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>FailureDomainLabels are the tag categories used for regions and zones in vSphere CSI driver and cloud controller.
See Cloud Native Storage: Set Up Zones in the vSphere CNS Environment
(<a href="https://docs.vmware.com/en/VMware-vSphere/6.7/Cloud-Native-Storage/GUID-9BD8CD12-CB24-4DF4-B4F0-A862D0C82C3B.html">https://docs.vmware.com/en/VMware-vSphere/6.7/Cloud-Native-Storage/GUID-9BD8CD12-CB24-4DF4-B4F0-A862D0C82C3B.html</a>)</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region is the tag category used for region on vSphere data centers and/or clusters.</p>
</td>
</tr>
<tr>
<td>
<code>zone</code></br>
<em>
string
</em>
</td>
<td>
<p>Zone is the tag category used for zones on vSphere data centers and/or clusters.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureStatus">InfrastructureStatus
</h3>
<p>
<p>InfrastructureStatus contains information about created infrastructure resources.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>vsphereConfig</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.VsphereConfig">
VsphereConfig
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>creationStarted</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>nsxtInfraState</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.NSXTInfraState">
NSXTInfraState
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.LoadBalancerClass">LoadBalancerClass
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.LoadBalancerConfig">LoadBalancerConfig</a>)
</p>
<p>
<p>LoadBalancerClass defines a restricted network setting for generic LoadBalancer classes.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the LB class</p>
</td>
</tr>
<tr>
<td>
<code>ipPoolName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>IPPoolName is the name of the NSX-T IP pool (must be set for the default load balancer class).</p>
</td>
</tr>
<tr>
<td>
<code>tcpAppProfileName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>TCPAppProfileName is the profile name of the load balaner profile for TCP</p>
</td>
</tr>
<tr>
<td>
<code>udpAppProfileName</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>UDPAppProfileName is the profile name of the load balaner profile for UDP</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.LoadBalancerConfig">LoadBalancerConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Constraints">Constraints</a>)
</p>
<p>
<p>LoadBalancerConfig contains the constraints for usable load balancer classes</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>size</code></br>
<em>
string
</em>
</td>
<td>
<p>Size is the NSX-T load balancer size (&ldquo;SMALL&rdquo;, &ldquo;MEDIUM&rdquo;, or &ldquo;LARGE&rdquo;)</p>
</td>
</tr>
<tr>
<td>
<code>classes</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.LoadBalancerClass">
[]LoadBalancerClass
</a>
</em>
</td>
<td>
<p>Classes are the defined load balancer classes</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImage">MachineImage
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.WorkerStatus">WorkerStatus</a>)
</p>
<p>
<p>MachineImage is a mapping from logical names and versions to provider-specific machine image data.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the logical version of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path is the path of the VM template.</p>
</td>
</tr>
<tr>
<td>
<code>guestId</code></br>
<em>
string
</em>
</td>
<td>
<p>GuestID is the optional guestId to overwrite the guestId of the VM template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImageVersion">MachineImageVersion
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImages">MachineImages</a>)
</p>
<p>
<p>MachineImageVersion contains a version and a provider-specific identifier.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
<p>Version is the version of the image.</p>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
<p>Path is the path of the VM template.</p>
</td>
</tr>
<tr>
<td>
<code>guestId</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>GuestID is the optional guestId to overwrite the guestId of the VM template.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImages">MachineImages
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>, 
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.RegionSpec">RegionSpec</a>)
</p>
<p>
<p>MachineImages is a mapping from logical names and versions to provider-specific identifiers.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the logical name of the machine image.</p>
</td>
</tr>
<tr>
<td>
<code>versions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImageVersion">
[]MachineImageVersion
</a>
</em>
</td>
<td>
<p>Versions contains versions and a provider-specific identifier.</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineTypeOptions">MachineTypeOptions
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>MachineTypeOptions defines additional VM options for an machine type given by name</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the machine type</p>
</td>
</tr>
<tr>
<td>
<code>memoryReservationLockedToMax</code></br>
<em>
bool
</em>
</td>
<td>
<em>(Optional)</em>
<p>MemoryReservationLockedToMax is flag to reserve all guest OS memory (no swapping in ESXi host)</p>
</td>
</tr>
<tr>
<td>
<code>extraConfig</code></br>
<em>
map[string]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ExtraConfig allows to specify additional VM options.
e.g. sched.swap.vmxSwapEnabled=false to disable the VMX process swap file</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.NSXTInfraState">NSXTInfraState
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureStatus">InfrastructureStatus</a>)
</p>
<p>
<p>NSXTInfraState holds the state of the infrastructure created with NSX-T</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>version</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>edgeClusterRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>transportZoneRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tier0GatewayRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>snatIPPoolRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>tier1GatewayRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>externalTier1Gateway</code></br>
<em>
bool
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>localeServiceRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>segmentRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>snatIPAddressAllocRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>snatRuleRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>snatIPAddress</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>segmentName</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>dhcpServerConfigRef</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">
Reference
</a>
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>advancedDHCP</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.AdvancedDHCPState">
AdvancedDHCPState
</a>
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.Networks">Networks
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureConfig">InfrastructureConfig</a>)
</p>
<p>
<p>Networks contains existing NSX-T network infrastructure to use.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>tier1GatewayPath</code></br>
<em>
string
</em>
</td>
<td>
<p>Tier1GatewayPath is the path of the existing NSX-T Tier-1 Gateway to use.</p>
</td>
</tr>
<tr>
<td>
<code>loadBalancerServicePath</code></br>
<em>
string
</em>
</td>
<td>
<p>LoadBalancerServicePath is the path of the existing NSX-T load balancer service assigned to the Tier-1 Gateway</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.Reference">Reference
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.NSXTInfraState">NSXTInfraState</a>)
</p>
<p>
<p>Reference holds a NSXT object reference managed with the NSX-T simplified / intent-based API</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>id</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
<tr>
<td>
<code>path</code></br>
<em>
string
</em>
</td>
<td>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.RegionSpec">RegionSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.CloudProfileConfig">CloudProfileConfig</a>)
</p>
<p>
<p>RegionSpec specifies the topology of a region and its zones.
A region consists of a Vcenter host, transport zone and optionally a data center.
A zone in a region consists of a data center (if not specified in the region), a computer cluster,
and optionally a resource zone or host system.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the region</p>
</td>
</tr>
<tr>
<td>
<code>vsphereHost</code></br>
<em>
string
</em>
</td>
<td>
<p>VsphereHost is the vSphere host</p>
</td>
</tr>
<tr>
<td>
<code>vsphereInsecureSSL</code></br>
<em>
bool
</em>
</td>
<td>
<p>VsphereInsecureSSL is a flag if insecure HTTPS is allowed for VsphereHost</p>
</td>
</tr>
<tr>
<td>
<code>nsxtHost</code></br>
<em>
string
</em>
</td>
<td>
<p>NSXTHost is the NSX-T host</p>
</td>
</tr>
<tr>
<td>
<code>nsxtInsecureSSL</code></br>
<em>
bool
</em>
</td>
<td>
<p>NSXTInsecureSSL is a flag if insecure HTTPS is allowed for NSXTHost</p>
</td>
</tr>
<tr>
<td>
<code>nsxtRemoteAuth</code></br>
<em>
bool
</em>
</td>
<td>
<p>NSXTRemoteAuth is a flag if NSX-T uses remote authentication (authentication done through the vIDM).</p>
</td>
</tr>
<tr>
<td>
<code>transportZone</code></br>
<em>
string
</em>
</td>
<td>
<p>TransportZone is the NSX-T transport zone</p>
</td>
</tr>
<tr>
<td>
<code>logicalTier0Router</code></br>
<em>
string
</em>
</td>
<td>
<p>LogicalTier0Router is the NSX-T logical tier 0 router</p>
</td>
</tr>
<tr>
<td>
<code>edgeCluster</code></br>
<em>
string
</em>
</td>
<td>
<p>EdgeCluster is the NSX-T edge cluster</p>
</td>
</tr>
<tr>
<td>
<code>snatIPPool</code></br>
<em>
string
</em>
</td>
<td>
<p>SNATIPPool is the NSX-T IP pool to allocate the SNAT ip address</p>
</td>
</tr>
<tr>
<td>
<code>datacenter</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Datacenter is the name of the vSphere data center (data center can either be defined at region or zone level)</p>
</td>
</tr>
<tr>
<td>
<code>datastore</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Datastore is the vSphere datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.</p>
</td>
</tr>
<tr>
<td>
<code>datastoreCluster</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>DatastoreCluster is the vSphere  datastore cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.</p>
</td>
</tr>
<tr>
<td>
<code>zones</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.ZoneSpec">
[]ZoneSpec
</a>
</em>
</td>
<td>
<p>Zones is the list of zone specifications of the region.</p>
</td>
</tr>
<tr>
<td>
<code>caFile</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>CaFile is the optional CA file to be trusted when connecting to vCenter. If not set, the node&rsquo;s CA certificates will be used. Only relevant if InsecureFlag=0</p>
</td>
</tr>
<tr>
<td>
<code>thumbprint</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Thumbprint is the optional vCenter certificate thumbprint, this ensures the correct certificate is used</p>
</td>
</tr>
<tr>
<td>
<code>dnsServers</code></br>
<em>
[]string
</em>
</td>
<td>
<em>(Optional)</em>
<p>DNSServers is a optional list of IPs of DNS servers used while creating subnets. If provided, it overwrites the global
DNSServers of the CloudProfileConfig</p>
</td>
</tr>
<tr>
<td>
<code>dhcpOptions</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.DHCPOption">
[]DHCPOption
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>DHCPOptions contains optional options for DHCP like Domain name, NTP server,&hellip;
If provided, it overwrites the global DHCPOptions of the CloudProfileConfig</p>
</td>
</tr>
<tr>
<td>
<code>machineImages</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.MachineImages">
[]MachineImages
</a>
</em>
</td>
<td>
<em>(Optional)</em>
<p>MachineImages is the list of machine images that are understood by the controller. If provided, it overwrites the global
MachineImages of the CloudProfileConfig</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.VsphereConfig">VsphereConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.InfrastructureStatus">InfrastructureStatus</a>)
</p>
<p>
<p>VsphereConfig holds information about vSphere resources to use.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>folder</code></br>
<em>
string
</em>
</td>
<td>
<p>Folder is the folder name to store the cloned machine VM</p>
</td>
</tr>
<tr>
<td>
<code>region</code></br>
<em>
string
</em>
</td>
<td>
<p>Region is the vSphere region</p>
</td>
</tr>
<tr>
<td>
<code>zoneConfigs</code></br>
<em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.ZoneConfig">
map[string]github.com/gardener/gardener-extension-provider-vsphere/pkg/apis/vsphere/v1alpha1.ZoneConfig
</a>
</em>
</td>
<td>
<p>ZoneConfig holds information about zone</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.ZoneConfig">ZoneConfig
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.VsphereConfig">VsphereConfig</a>)
</p>
<p>
<p>ZoneConfig holds zone specific information about vSphere resources to use.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>datacenter</code></br>
<em>
string
</em>
</td>
<td>
<p>Datacenter is the name of the data center</p>
</td>
</tr>
<tr>
<td>
<code>computeCluster</code></br>
<em>
string
</em>
</td>
<td>
<p>ComputeCluster is the name of the compute cluster. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>resourcePool</code></br>
<em>
string
</em>
</td>
<td>
<p>ResourcePool is the name of the resource pool. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>hostSystem</code></br>
<em>
string
</em>
</td>
<td>
<p>HostSystem is the name of the host system. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>datastore</code></br>
<em>
string
</em>
</td>
<td>
<p>Datastore is the datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified</p>
</td>
</tr>
<tr>
<td>
<code>datastoreCluster</code></br>
<em>
string
</em>
</td>
<td>
<p>DatastoreCluster is the datastore  cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified</p>
</td>
</tr>
<tr>
<td>
<code>switchUuid</code></br>
<em>
string
</em>
</td>
<td>
<p>SwitchUUID is the UUID of the virtual distributed switch the network is assigned to (only needed if there are multiple vds)</p>
</td>
</tr>
</tbody>
</table>
<h3 id="vsphere.provider.extensions.gardener.cloud/v1alpha1.ZoneSpec">ZoneSpec
</h3>
<p>
(<em>Appears on:</em>
<a href="#vsphere.provider.extensions.gardener.cloud/v1alpha1.RegionSpec">RegionSpec</a>)
</p>
<p>
<p>ZoneSpec specifies a zone of a region.
A zone in a region consists of a data center (if not specified in the region), a computer cluster,
and optionally a resource zone or host system.</p>
</p>
<table>
<thead>
<tr>
<th>Field</th>
<th>Description</th>
</tr>
</thead>
<tbody>
<tr>
<td>
<code>name</code></br>
<em>
string
</em>
</td>
<td>
<p>Name is the name of the zone</p>
</td>
</tr>
<tr>
<td>
<code>datacenter</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Datacenter is the name of the vSphere data center (data center can either be defined at region or zone level)</p>
</td>
</tr>
<tr>
<td>
<code>computeCluster</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ComputeCluster is the name of the vSphere compute cluster. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>resourcePool</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>ResourcePool is the name of the vSphere resource pool. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>hostSystem</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>HostSystem is the name of the vSphere host system. Either ComputeCluster or ResourcePool or HostSystem must be specified</p>
</td>
</tr>
<tr>
<td>
<code>datastore</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>Datastore is the vSphere datastore to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.</p>
</td>
</tr>
<tr>
<td>
<code>datastoreCluster</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>DatastoreCluster is the vSphere  datastore cluster to store the cloned machine VM. Either Datastore or DatastoreCluster must be specified at region or zones level.</p>
</td>
</tr>
<tr>
<td>
<code>switchUuid</code></br>
<em>
string
</em>
</td>
<td>
<em>(Optional)</em>
<p>SwitchUUID is the UUID of the virtual distributed switch the network is assigned to (only needed if there are multiple vds)</p>
</td>
</tr>
</tbody>
</table>
<hr/>
<p><em>
Generated with <a href="https://github.com/ahmetb/gen-crd-api-reference-docs">gen-crd-api-reference-docs</a>
</em></p>
