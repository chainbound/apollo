# Apollo
> **Query**, **transform**, **filter** and **save** EVM based chaindata using a simple schema

![apollo-diagram drawio](./docs/apollo-flow.png)

## Documentation
For detailed documentation, visit [apollo.chainbound.io](https://apollo.chainbound.io).

## Installation
```bash
go install github.com/chainbound/apollo
```

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

`$HOME/.config/apollo/config.yml` will be configured with some standard chains and public RPC APIs. These will not do
for most queries, and we recommend either using your own node, or getting one with a node provider
like Alchemy or Chainstack.

### Schema
`$HOME/.config/apollo/schema.hcl` is configured with a default schema (below) that you can try out, but for a more in depth
explanation visit the [schema documentation](https://apollo.chainbound.io/schema/intro) or check out 
some [examples](https://apollo.chainbound.io/schema/schema-examples).
```hcl
start_time = format_date("02-01-2006 15:04", "25-05-2022 12:00")
end_time = now

variables = {
  b = upper("eth_buy")
  s = upper("eth_sell")
}

// query defines the name of your query -> name of your output files and SQL tables
query usdc_eth_swaps {
  // Each query can have a different chain
  chain = "arbitrum"

  contract {
    address = "0x905dfCD5649217c42684f23958568e533C711Aa3"
    abi = "unipair.abi.json"
    // Listen for events
    event Swap {
      // The outputs we're interested in, same way as with methods.
      outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
    }

    // "transform" blocks are at the contract-level
    transform {
      usdc_sold = parse_decimals(amount1In, 6)
      eth_sold = parse_decimals(amount0In, 18)

      usdc_bought = parse_decimals(amount1Out, 6)
      eth_bought = parse_decimals(amount0Out, 18)

      buy = amount0Out != 0
    }
  }

  filter = [
    eth_bought != 0
  ]

  // Besides the normal context, the "save" block for events provides an additional
  // variable "tx_hash". "save" blocks are at the query-level and have access to variables
  // defined in the "transform" block
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    // Example: we want to calculate the price of the swap.
    // We have to make sure we don't divide by 0, so we use the ternary operator.
    swap_price = eth_bought != 0 ? (usdc_sold / eth_bought) : (usdc_bought / eth_sold)
    direction = buy ? b : s
    size_in_udsc = eth_bought != 0 ? usdc_sold : usdc_bought
  }
}
```

### Running
**Important**: running `apollo` with the default parameters will send out a lot of requests, and your node provider might rate limit you.
Please check the [rate limiting](https://apollo.chainbound.io/getting-started#rate-limiting) section in the documentation. You can set
the `--rate-limit` option to something low like 20 to start.

#### Realtime mode
After defining the schema, run
```bash
apollo --realtime --stdout
```
In the case of events, this will listen for events in real-time and save them in your output option.
In the case of methods, you will have to define one of the `interval` parameters,
and `apollo` will run that query at every interval.

#### Historical mode
After defining the schema with `start`, `end` and `interval` parameters, just run
```bash
apollo --stdout
```
The default mode is historical mode.

## Output
There are 3 output options:
* `stdout`: this will just print the results to your terminal.
* `csv`: this will save your output into a csv file. The name of your file will be the name of your `query`. The other columns
will be made up of what's defined in the `save` block.
* `db`: this will save your output into a Postgres SQL table, with the table name matching your `query` name. The settings are defined in `config.yml` in your `apollo` config directory.
