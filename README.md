# Apollo
> Program for easily querying and collecting EVM chaindata based on a schema.

## Introduction
`apollo` is a program for querying and collecting EVM chaindata based on a [schema](#schema). Chaindata in this case is
one of these things:
* Contract methods
* Contract events

It can run in 2 modes:
* **Historical mode**: here we define a block range and interval for method calls, or just a block range for events. `apollo`
will run through all the blocks and collect the data we've defined in the schema. This can be useful for backtesting etc.
* **Realtime mode**: we define a time interval at which we want to collect data. This is useful for building a backend database
for stuff like dashboards for live strategies.

## Usage
### Setting up
First, generate the config directory and files:
```
apollo init
```
This will generate the configuration files (`config.yml` and `schema.yml`) and put it into your configuration
directory, which will either be `$XDG_CONFIG_HOME/apollo` or `$HOME/.config/apollo`. This is the directory
in which you have to configure `apollo`, and it's also the directory where `apollo` will try to find the specified
contract ABIs.

### Running
* **Realtime mode**
In realtime mode, we only have to define the interval if we're doing a method calling schema (in seconds) and the chain, 
and an optional output option (either `--csv`, `--db` or `--stdout`)

*Examples* 

* Run a method calling schema every 5 seconds in realtime on Arbitrum, and save the results in a csv
```
apollo --realtime --interval 5 --csv --chain arbitrum
```
* Run an event collecting schema in realtime on Ethereum, save the results in a database
```
apollo --realtime --db --chain ethereum
```

* **Historical mode**
In historical mode, we define the start and end blocks, the chain, the interval (when we're doing method calls),
and an optional output option.

*Examples* 

* Run a method calling schema every 100 blocks with a start and end block on Arbitrum, and save the results in a DB and a csv
```
apollo --start-block 1000000 --end-block 1200000 --interval 100 --csv --db --chain arbitrum 
```
* Run an event collecting schema over a range of blocks on Polygon, and output the results to `stdout`
```
apollo --start-block 1000000 --end-block 1500000 --stdout --chain polygon 
```

**All Options**
```
GLOBAL OPTIONS:
   --realtime, -R                 Run apollo in realtime (default: false)
   --db                           Save results in database (default: false)
   --csv                          Save results in csv file (default: false)
   --stdout                       Print to stdout (default: false)
   --interval BLOCKS, -i BLOCKS   Interval in BLOCKS or SECONDS (realtime: seconds, historic: blocks) (default: 0)
   --start-block value, -s value  Starting block number for historical analysis (default: 0)
   --end-block value, -e value    End block number for historical analysis (default: 0)
   --chain value, -c value        The chain name
   --rate-limit LEVEL             Rate limit LEVEL, from 1 - 5 (default: 0)
   --help, -h                     show help (default: false)
```

## Schema
The schema is in the form of a YAML file which defines the data we're interested in. The top-level elements are `chain` and `contracts`.
`contracts` will define the data we want. There are some annotated examples below.
### Methods Example
```yaml
# Define the chain to run on
chain: arbitrum

# The contracts to populate tables for
contracts:
    # Address of the contract
  - address: 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8
    # The name of the contract (will be the table name in the DB and name of the CSV file)
    name: usdc
    # ABI file (in ~/.config/apollo)
    abi: erc20.abi.json
    # Methods we want to call (as a list)
    methods:
      - name: balanceOf
        # Arguments according to the ABI
        inputs:
          _owner: 0xe1Dd30fecAb8a63105F2C035B084BfC6Ca5B1493
        # The outputs we want according to the ABI.
        outputs:
          - balance

      - name: totalSupply
        # No args in this case
        outputs:
          # the totalSupply() method does not have named outputs, so
          # this will dynamically name the output parameter 'supply'
          - supply

  - address: 0x639Fe6ab55C921f74e7fac1ee960C0B6293ba612
    name: eth_chainlink_feed
    abi: feed.abi.json
    methods:
      - name: latestRoundData
        # This method has no inputs, so we can leave "args" out

        # The method has multiple outputs, here we specify which ones
        # we're interested in.
        outputs:
          - answer
          - roundId
          - updatedAt
```
### Events Example
```yaml
# Define the chain to run on
chain: arbitrum

# The contracts to populate tables for
contracts:
    # Address of the contract on corresponding chain (Arbitrum)
  - address: 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8
    # The name of the contract (will be the table name in the DB and name of CSV file)
    name: usdc_transfer_events
    # ABI file (in ~/.config/apollo)
    abi: erc20.abi.json
    events:
      - name: Transfer
        outputs:
          - from
          - to
          - value
```

## Output
There are 3 output options:
* `stdout`: this will just print the results to your terminal.
* `csv`: this will save your output into a csv file. The name of your file corresponds to the `name` field of your contract schema definition. The other columns are going to be the inputs and outputs fields, also with the names corresponding to the schema.
* `db`: this will save your output into a Postgres SQL table. The settings are defined in `config.yml` in your `apollo`
config directory.