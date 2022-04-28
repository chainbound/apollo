# Apollo
> Program for easily querying and collecting EVM chaindata based on a schema.

## To Do
- [ ] How to best generate dynamic golang code (based on schema)
  - [ ] Look at custom struct tags
  - [ ] Should probably not generate golang code but just ABI pack values into transaction input field
- [x] How to best generate SQL DDL based on schema
  - Should we use an ORM package or just plain SQL?
  - For now, just plain SQL will do
- [x] Simplify schema parser
  - [x] Upgrade schema to yaml

## Program Execution
1. DDL is generated based on the schema
2. This DDL is then executed on the DB and the relevant tables are created
3. Ethereum call messages are generated based on the schema and the ABI
4. The main program loop starts, executing the call messages in an interval
5. The values are written to the tables

## Example Schema
```yml
# Define the chain to run on
chain: arbitrum

# The contracts to populate tables for
contracts:
    # Address of the contract on corresponding chain (Arbitrum)
  - address: 0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8
    # The name of the contract (will be the table name in the DB)
    name: usdc
    # ABI file path
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

## Notes
* Maybe in the future we could bypass the schema and just create an SQL chain query directly.
Think of how Dune Analytics does it