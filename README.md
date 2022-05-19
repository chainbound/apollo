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
This will generate the configuration files (`config.yml` and `schema.hcl`) and put it into your configuration
directory, which will either be `$XDG_CONFIG_HOME/apollo` or `$HOME/.config/apollo`. This is the directory
in which you have to configure `apollo`, and it's also the directory where `apollo` will try to find the specified
contract ABIs.

### Running
* **Realtime mode**
In realtime mode, we only have to define the interval if we're doing a method calling schema (in seconds) and the chain, 
and an optional output option (either `--csv`, `--db` or `--stdout`). See the [Output](##Output) for more info on that.

*Examples* 

* Run a method calling schema every 5 seconds in realtime on Arbitrum, and save the results in a csv
```
apollo --realtime --interval 5 --csv
```
* Run an event collecting schema in realtime on Ethereum, save the results in a database
```
apollo --realtime --db
```

* **Historical mode**
In historical mode, we define the start and end blocks, the chain, the interval (when we're doing method calls),
and an optional output option.

*Examples* 

* Run a method calling schema every 100 blocks with a start and end block on Arbitrum, and save the results in a DB and a csv
```
apollo --start-block 1000000 --end-block 1200000 --interval 100 --csv --db
```
* Run an event collecting schema over a range of blocks on Polygon, and output the results to `stdout`
```
apollo --start-block 1000000 --end-block 1500000 --stdout
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
The schema is in the form of a DSL implemented with [HCL](https://github.com/hashicorp/hcl) to define the data
we're interested in. This means that basic arithmetic operations and ternary operators
for control flow are supported by default. The top-level elements are `chain` and `contract`.
In the `contract` block we provide the ABI file, along with which `methods` or `events` we want to get the data for.

In the case of a `method`, we first define `inputs` and `outputs`. For an `event`, it's only `outputs`.
The names of the methods, events, inputs and outputs should correspond exactly to what's in the provided
ABI file.

The last block in `contract` is the `save` block. In this block we can do some basic transformations
before saving our output, and it provides access to variables and functions that we might need. 

### Save Context
Any `input` or `output` is provided as a variable by default.
Other variables available are:
* `timestamp`
* `blocknumber`
* `contract_address`

And for `events`:
* `tx_hash`

The available functions are:
* `lower(str)`
* `upper(str)`
* `parse_decimals(raw, decimals)`

Below are some annotated examples to help you get started. There are some more examples in the [docs](docs/schema-examples.md).
### Methods Example
```hcl
// Define the chain to run on
chain = "arbitrum"

contract usdc_eth_reserves "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  // Will search in the Apollo config directory
  abi = "unipair.abi.json"

  // Call methods
  method getReserves {
    // These are the outputs we're interested in. They are available 
    // to transform as variables in the "save" block below. Outputs should
    // be provided as a list.
    outputs = ["_reserve0", "_reserve1"]
  }

  // The "save" block will give us access to more context, including variables
  // like "timestamp", "blocknumber", "contract_address", and any inputs or outputs
  // defined earlier.
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    eth_reserve = parse_decimals(_reserve0, 18)
    usdc_reserve = parse_decimals(_reserve1, 6)

    // Example: we want to calculate the mid price from the 2 reserves.
    // Cannot reuse variables that are defined in the same "save" block.
    // We have to reuse variables that were defined in advance, i.e.
    // in "inputs" or "outputs"
    mid_price = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)

  }
}
```
### Events Example
```hcl
// Define the chain to run on
chain = "arbitrum"

contract usdc_to_eth_swaps "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  // Will search in the Apollo config directory
  abi = "unipair.abi.json"

  // Listen for events
  event Swap {
    // The outputs we're interested in, same way as with methods.
    outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
  }


  // Besides the normal context, the "save" block for events provides an additional
  // variable "tx_hash". This is the transaction hash of the originating transaction.
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    // Example: we want to calculate the price of the swap.
    price = amount0Out != 0 ? (parse_decimals(amount1In, 6) / parse_decimals(amount0Out, 18)) : (parse_decimals(amount1Out, 6) / parse_decimals(amount0In, 18))
    dir = amount0Out != 0 ? upper("buy") : upper("sell")
    size = amount1In != 0 ? parse_decimals(amount1In, 6) : parse_decimals(amount1Out, 6)
  }
}
```

## Output
There are 3 output options:
* `stdout`: this will just print the results to your terminal.
* `csv`: this will save your output into a csv file. The name of your file corresponds to the `name` field of your contract schema definition. The other columns are going to be the inputs and outputs fields, also with the names corresponding to the schema.
* `db`: this will save your output into a Postgres SQL table. The settings are defined in `config.yml` in your `apollo`
config directory.