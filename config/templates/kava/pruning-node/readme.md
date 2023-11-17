The `pruning-node` can optionally run alongside the validator.

The pruning node will use the same configuration as the base template used to start kvtool
so that it runs the same version with same genesis.
The only difference is that it is stripped of its validator keys & is given a new node_key.

The docker image is automatically replaced in the docker-compose.yaml file based on the kava template used.

**Note**: The node is "pruning" in name only. It's pruning configuration is controlled by the settings in app.toml.

**Example**
```
kvtool t bootstrap --config.kavaTemplate=v0.24 --pruning
```
The above runs a network on the v0.24 template with one validator & a pruning node.
