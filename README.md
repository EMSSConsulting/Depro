# Depro Deployment Manager
**Simplified cluster application deployment using Consul**

Depro aims to provide the following features to assist in the deployment of versioned
software in a cluster environment in which Consul is used.

1. Scale deployment time vs. number of targets better than linearly.
2. Provide strong guarantees regarding the version of code running on your cluster.
3. Allow rollback/rollforward of code while guaranteeing **2**.
4. Scale disk cleanup time vs. number of deployment targets better than linearly.
5. Automatically bootstrap new deployment targets with minimal human interaction.

The above features and guarantees were born of trying to manage a strongly agile
project in which multiple deployments were conducted each day across a cluster
large enough to make manual intervention expensive and sequential deployments
laboriously time consuming.

## Addressing Features
Depro addresses the features/guarantees listed above by approaching the problem
in three phases. This is all centrally administered through the Consul distributed
key-value store.

### Consul Tree
```
 + <prefix>
   - current = <version>
   + <version>
     - <node> = "busy" | "ready"
```

### Phase 1 - Artifact Deployment
The artifact deployment phase involves stashing the build artifacts on a server
available to all of the deployment targets and adding an entry to the Consul
`<prefix>` folder.

Agents running on each deployment target watch the `<prefix>` folder for changes
and will automatically start Phase 2 when they see a change.

### Phase 2 - Target Bootstrapping
Agent bootstrapping is the process of updating a deployment target's state to
ensure it is prepared to run a specific version of the code. The agent maintains
a directory structure mirroring the Consul `<prefix>` tree and will perform
differential comparisons against that tree to decide which operations to perform.

#### Version Added
When a version is added, the agent will pull `$version.tar` from the configured
artifact server and untar it into the `$version` folder. In addition to this, the
agent will set the value of `<prefix>/<version>/<node>` to "busy" when it first
notices the new version, and update it to "ready" once the version has been successfully
extracted.

This allow another Consul client to observe the `<prefix>/<version>` key-prefix
to determine when each node completes its deployment of the given version.

#### Version Removed
If a version has been removed from the Consul tree, the agent will remove the
corresponding version directory as well as the `<prefix>/<version>/<node>` key.

This allows an observing client to determine when a version has been purged from
each of the nodes in the cluster.

It is important to be aware that if the removed version matches the current version,
agents will assume that the cluster server has been erroneously modified and will
attempt to repair the version tree by re-registering their entries. As such, it is
important that you checkout a new version prior to removing the current one.

#### Current Version Changed
Finally, when the current version key is updated in Consul, this change is compared
to the current version in the agent's copy. If the version matches, no action is
taken. If the version differs, the agent will update its local version file and
run the configured checkout command.

To prevent race conditions, the version changed handler is only ever processed after
all version additions and removals complete. This helps ensure that under initial
bootstrapping, code is only checked out once it becomes available.
In addition to this, the agent's local version will only be updated if it has a
copy of the referenced version on its local storage.

### Phase 3 - Checkout
Once each of the target nodes has been bootstrapped, the client may update the current
version entry. This will instruct each agent to transition to the specified version
of the code.

It is important that you only update the current version entry once all nodes have
reported that they are in the "ready" state for the referenced version. Failure to
do so will result in the guarantee of consistency being violated should some nodes
still be in the process of checkout out the version.
