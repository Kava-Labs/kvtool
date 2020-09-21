# Architecture Decision Record

## Goal

We want a tool that can set up testnets of our various services in several different configurations. For example:

- a kvd node, bnbchain node (with rest servers), and the deputy for integration testing the web app
- a bare kvd node built from unmerged branches for helping with PR reviews
- a local kvd node with genesis that closely mirrors mainnet to test out upcoming gov proposals
- a kvd node and oracle to test out CDP top up bot
- etc

## Proposal

Use docker and docker-compose to manage the running of any local testnets/rest-servers/deputies/bots.

Create go packages and/or cli tools only to generate a set of config files for docker-compose.

### What this would look like

1) Stage 1 is to generate a config

	- could be done manually, or with aid of cli tools
	- output is a directory containing a `docker-compose.yaml`. The file can reference config files in folders within that directory.
	- example:
	```
		generated_config/
		|- kava/
			|- .kvd/
				|- config.toml
				|- genesis.json
		|- binance/
			|- .bnbchaind/
				|- config.toml
				|- genesis.json
			|- .bnbcli/
		|- deputy/
			|- config.json
		|- docker-compose.yaml
	```

1) Stage 2 is to run it:
	- `docker-compose --file output_config/docker-compose.yaml up`

### Advantages

- config can be inspected manually before running
- config can be edited (or created) manually for ad-hoc testnets, without needing to change the config generation code
- reduces scope of the tools to only output static config, and not manage running processes

### Disadvantages

- requires working knowledge of docker / docker-compose
- services need to be packaged into docker containers (although for the web app, it could run outside docker while interacting with kava/bnb rest servers running in docker)
- there might be need to send in txs after a testnet launches to create CDPs, auctions, etc. There is not obvious place for this type of configuration to live in this framework

### Generating Config

I think the value of this tool comes from assembling configurations for different services that work together.

Currently there are a load of manually created files in `config_templates` which are copied and lightly edited when `kvtool testnet` runs. This works for now but in the future moving all of the data into go packages will be more scalable.

## Resources

### docker golang APIs

The docker cli is written in go. Docker compose is written in python. Both communicate with the local docker server to run or build or do stuff with containers.

The docker-compose yaml format is supported in go here: https://github.com/docker/cli/tree/master/cli/compose. It allows you to work with compose files, but not do things like run docker-compose up from within go. That was supported by this module: https://github.com/docker/libcompose but is no longer maintained.

### editing docker-compose.yaml

We could use the types from github.com/docker/cli/cli/compose to unmarshal docker-compose.yaml files. However it has no `go.mod` and a lot of dependencies (kept in the vendor directory, with versions listed in `vendor.conf`) so importing is awkward.

	bz, err := ioutil.ReadFile("docker-compose.yaml")
	if err != nil {
		return err
	}
	mapping, err := loader.ParseYAML(bz)

	preloadConfig := types.ConfigDetails{
		Version:    "", // ?
		WorkingDir: "", // ?
		ConfigFiles: []types.ConfigFile{{
			Filename: "docker-compose.yaml", // ?
			Config:   mapping,
		}},
		Environment: nil, // ?
	}
	composeConfig, err := loader.Load(preloadConfig)
	if err != nil {
		return err
	}
