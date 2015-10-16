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

## Usage

### Deployment Tool
The deployment tool is designed to make deploying a version of your application
a simple process. Once you have made your build artifacts available, simply run
the deployment tool to have your cluster deploy and rollout the new version.

```sh
depro deploy 585ecfabf5b41bae1db7bd566ce984d77568987d -prefix=api/version -nodes=3
```

When running the deploy tool, you should ensure that you provide a value for the
prefix and nodes parameters - as these will dictate which cluster receives the
version as well as the minimum number of nodes required to acknowledge the deployment
before it will take place.

### Deployment Agent
Depro is run as an agent on each of your deployment targets, on which it will
manage the defined deployment path based on the contents of your Consul
key value store.

The agent watches for changes to Consul's entries and will update the local
system to match the state presented in Consul. This state can either be managed
manually, through custom scripts or using the deployment tool built into Depro.

```sh
depro agent -config-dir=/etc/depro/
```

```json
{
    "name": "workerNode1",
    "server": "localhost:8500",
    "datacenter": "dc1",
    "deployments": [
        {
            "id": "api",
            "path": "/data/deploy/api/",
            "prefix": "api/version",
            "shell": "bash",
            "deploy": [
                "wget -O - http://artifacts.myapp.com/api/$VERSION.tar.gz | tar zxf - || exit 1"
                "rackadmin register $DEPLOYMENT_PATH $VERSION"
            ],
            "rollout": [
                "rackadmin checkout $DEPLOYMENT_PATH $VERSION"
            ],
            "clean": [
                "rackadmin clean $DEPLOYMENT_PATH $VERSION"
            ]
        },
        {
            "id": "website",
            "path": "/data/deploy/website/",
            "prefix": "website/version",
            "shell": "bash",
            "deploy": [
                "wget -O - http://artifacts.myapp.com/website/$VERSION.tar.gz | tar zxf - || exit 1"
                "rackadmin register $DEPLOYMENT_PATH $VERSION"
            ],
            "rollout": [
                "rm $DEPLOYMENT_PATH/live",
                "ln -s $DEPLOYMENT_PATH/$VERSION $DEPLOYMENT_PATH/live",
                "systemctl reload nginx"
            ]
        }
    ]
}
```

## Design
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
