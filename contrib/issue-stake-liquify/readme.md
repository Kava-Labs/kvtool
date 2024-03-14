# `issue-stake-liquify`

This script is used to delegate kava across a set of validators on a testnet.

From the configuration input (see below) it will:
* issue kava from the dev wallet to each delegator
* delegate the kava across the validators by a designated distribution
* turn the staked kava into liquid bkava for each validator
* deposit the bkava denoms into earn

## Initial Setup

Configure a `.env` for what network to connect to and the mnemonics for necessary accounts. See [.example.env](./.example.env) for details on required environment variables.

The script is run by passing the configuration JSON via stdin.

Via redirection:
```bash
go run main.go < configuration.json
```

Via pipes:
```bash
cat configuration.json | go run main.go
```

## Configuration JSON

### Defining Validators
A minimal working example input looks like the following:
```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ]
}
```

By default, this will delegate an equal amount of Kava to each validator from one delegator account.

`validators` - a required list of validators to delegate to. The only key necessary is the `operator_address`, the `kavavaloper-` address of the validator.

Other fields are ignored which means it works to pass in a validator list from the API:
```bash
curl -s http://localhost:1317/cosmos/staking/v1beta1/validators | go run main.go
```

Or from a non-zero height state export:
```bash
jq .app_state.staking.validators export-1764346.json | go run main.go
```

### Equal Distribution

The input accepts an optional `delegations` field.

When not defined, it defaults to the following:
```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ],
  // each item in `delegations` is an account that will delegate to above validators
  // if empty or undefined, defaults to 1 account with equal distribution
  "delegations": [
    // the default delegation.
    {
      // "equal" distribution delegates base_amount to each validator
      "distribution": "equal",
      "base_amount": "1_000_000_000"
    }
  ]
}
```

`distribution` - the delegation distribution strategy. either `"equal"` or `"custom"` (see below).
`base_amount` - the amount in ukava of each delegation

When no `delegations` object is defined, it will use an `"equal"` distribution with a `base_amount` set to the `DEFAULT_BASE_AMOUNT` environment variable (which falls back to 1,000 KAVA if undefined).

The above delegates a total of 2,000 KAVA, 1,000 KAVA to each validator.

**example: Equal delegation, custom amount**
Delegate an equal amount of KAVA to all validators with a custom amount by defining the `delegations` object with the desired `base_amount`.

Here, we delegate 1M KAVA to all validators running on testnet:
```bash
curl -s https://api.testnet.kava.io/cosmos/staking/v1beta1/validators |
  jq '{
    validators: .validators,
    delegations: [{ distribution: "equal", base_amount: "1_000_000_000_000" }]
  }' |
  go run main.go
```

Another option is to change the `DEFAULT_BASE_AMOUNT` in the `.env`.

**example: Equal delegation from multiple accounts**
Each item in `delegations` defines how a different delegator accounts will delegate KAVA to the validators.

```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ],
  "delegations": [
    { "distribution": "equal", "base_amount": "1_000_000_000_000"},
    { "distribution": "equal", "base_amount": "5_000_000_000"},
  ]
}
```

The above configuration defines two accounts with the following delegations:
* Account 0 delegates 1M KAVA to Validator 0 & Validator 1
* Account 1 delegates 5k KAVA to Validator 0 & Validator 1

Accounts are generated from the same mnemonic with different account indexes in the HD path.

### Custom Distributions

The configuration input allows for arbitrarily weighted distributions when `distribution` is `"custom"`.

```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ],
  "delegations": [
    {
      // "custom" distribution delegates weight * base_amount to the corresponding validator
      "distribution": "custom",
      "base_amount": "100_000_000_000",
      "weights": [9, 1]
    }
  ]
}
```

The above delegates 1M KAVA:
* 900k KAVA to Validator 0
* 100k KAVA to Validator 1

**example: Custom delegation, only top `n` validators**
The configuration input will ignore any missing or extra `weights` provided to `"custom"` distributions. This means that you can delegate only a top `n` validators of a set, even if that many validators are not provided:
```bash
curl -s https://api.testnet.kava.io/cosmos/staking/v1beta1/validators |
  jq '{
    validators: .validators,
    delegations: [{
      distribution: "custom",
      base_amount: "1_000_000_000_000",
      weights: [1, 1, 1]
    }]}' | go run main.go
```

The above makes a delegation of 3M KAVA; 1M to each of the first THREE validators (assuming they exist), and no delegation to any others.

### Arbitrary Complexity!

Mix and match to your heart's content:
```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ],
  "delegations": [
    { "distribution": "equal", "base_amount": "1_000_000_000" },
    {
      "distribution": "custom",
      "base_amount": "100_000_000_000",
      "weights": [8, 2]
    }
  ]
}
```

The above creates the following delegations:
* Account 0 delegates 2,000 KAVA total; 1,000 KAVA to each validator
* Account 1 delegates 1M KAVA total; 800k to validator 0 and 200k to validator 1

### Spam Delegations

If you don't need an explicit delegation distribution and just want a bunch of state created, you can define a `spam_delegations` object that will be used to create the delegations.
```json
{
  "validators": [
    { "operator_address": "kavavaloper1xcgtffvv2yeqmgs3yz4gv29kgjrj8usxrnrlwp" },
    { "operator_address": "kavavaloper1w66m9hdzwgd6uc8g93zqkcumgwzrpcw958sh3s" }
  ],
  "spam_delegations": {
    "count": 100,
    "min_amount": "100_000_000",
    "max_amount": "1_000_000_000_000"
  }
}
```

The above creates 100 delegations from 100 different accounts. The delegations will be of random amounts between 100 KAVA and 1M KAVA. Each account's delegation cycles through the validators, so in the above, 50 delegations will be to validator 0 and 50 will be to validator 1.

**example: spam all validators with 10,000 delegations between 100 & 1M KAVA**
```bash
curl -s https://api.testnet.kava.io/cosmos/staking/v1beta1/validators |
  jq '{
    validators: .validators,
    spam_delegations: [{
      count: 10000,
      min_amount: "100_000_000",
      max_amount: "1_000_000_000_000",
    }]}' | go run main.go
```
