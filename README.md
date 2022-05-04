# Apollo
> Program for easily querying and collecting EVM chaindata based on a schema.

## Usage
First, generate the config directory:
```
apollo init
```
This will output where the configuration files were written to. **This path is where you will have
to save the ABIs for the contracts you want to query.**

Next up, put the ABI in your config dir and modify the schema to fit your requests.
### Examples
* Run a schema every 5 seconds in realtime on Arbitrum, and save the results in a csv
```
apollo --realtime --interval 5 --csv --chain arbitrum
```

* Run a schema every 100 blocks with a start and end block on Arbitrum, and save the results in a DB and a csv
```
apollo --start-block 1000000 --end-block 1200000 --interval 100 --csv --db --chain arbitrum 
```

**All Options**
```
NAME:
   apollo - Run the chain analyzer

USAGE:
   apollo [global options] command [command options] [arguments...]

COMMANDS:
   help, h  Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --realtime, -R                 Run apollo in realtime (default: false)
   --db                           Save results in database (default: false)
   --csv                          Save results in csv file (default: false)
   --stdout                       Print to stdout (default: false)
   --interval BLOCKS, -i BLOCKS   Interval in BLOCKS or SECONDS (realtime: seconds, historic: blocks) (default: 0)
   --start-block value, -s value  Starting block number for historical analysis (default: 0)
   --end-block value, -e value    End block number for historical analysis (default: 0)
   --chain value, -c value        The chain name
   --help, -h                     show help (default: false)
```

## Program Execution
1. DDL is generated based on the schema
2. This DDL is then executed on the DB and the relevant tables are created
3. Ethereum call messages are generated based on the schema and the ABI
4. The main program loop starts, executing the call messages in an interval
5. The values are written to the tables

## Schema
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
        args:
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