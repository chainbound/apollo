# Roadmap

## First release
- [ ] **v1.0.1-beta**
  - [x] Logging with log levels
  - [x] Improved stdout output
  - [x] Timestamps for setting start, end and interval options
  - [x] Review concurrency model
        - The problem with the current concurrency model is that when the max number of workers is reached, it waits until
        **all** goroutines in that batch finish before starting another batch. This is not what we want, since some goroutines
        can take disproportionate amounts of time and thus block the program from collecting more data.
        - Fixed with "go.uber.org/ratelimit" package. We can now define a max number of requests per second.
  - [x] Working DB output
  - [x] Ability to call methods when logs occur, and not just at a random interval. This would make it easier for some use cases. The DSL syntax will be defining a method block inside of an event block. NOTE: method calls are at the block level (happen at the 
  **end** of a block, while events are on the transaction level)
  - [ ] Filter lists
      - Example: when one of these evaluates to `false`, don't proceed to `transform` or `save`. This should also be a top-level
      block. It's like an SQL `WHERE` clause.
      ```hcl
        filter = [
            _reserve0 != 0,
            _reserve1 != 0
        ]
      ```
  - [x] Standalone events (not emitted from a certain contract)
        - Example: if we want **every** ERC20 transfer, we don't want to define the `event` in a `contract` block but as a top-level block.
  - [ ] Native asset balances with a `balance` block
  - [x] `transform` blocks, which are contract level blocks to define and transform variables to be used later in the top-level save blocks 
  - [x] `save` block should be a top-level block so that we can do cross-contract operations
      - This only works for results that happen at the same time, i.e. method calls
  - [x] Variables
  - [ ] Request batching
  - [ ] Multi-chain global events
  - [ ] Multi-chain native balances 
  - [ ] ETH + ARBI node deployment 
  - [ ] Major update of Docs
  - [x] Add more context variables: `tx_index`, `block_hash`
  - [ ] In Documentation add 'advanceed features'
  - [ ]   division by zero (`?:` operator)
  - [ ]   `block_offset`
  - [ ]   `for_loops`
  - [ ] Refactor + error handling and reliability

- [ ] **v1.1.0-beta**
  - [ ] Transaction monitoring
      - You would be able to filter historical transactions based on certain predicates: value thresholds, sender and receiver addresses, gas prices and amounts, or certain method calls or inputs.
  - [ ] Mempool monitoring
      - You would be able to monitor mempool transactions and save them based on a predicate. Same as above. 
  - [ ] Different stream output option for latency-sensitive operations (like mempool monitoring): i.e. Websocket, SSE 
      - Latency sensitive operations would probably also need different evaluation options. I think evaluating everything in the save block might take some time, would need to benchmark that. An option is to just not have a save block and stream everything as-is, let the application take care of decoding.
  - [ ] JSON output
  - [ ] Events: `tx_sender` context variable
  - [ ] Algorithm for determining `event` range (start big, if we get error, read range and modify)
  - [ ] Generalized SQL output (MySQL, SQL Server)
  - [ ] Aggregation operations like group by, sum, avg
  - [ ] Unverified methods and events
  - [ ] Cross-chain address monitoring
  - [ ] Loops (multiple inputs => call method for every input)

## Later
- [ ] **v1.2.0-beta**
  - [ ] Transaction simulation at certain times / events
  - [ ] Look into Apache Spark and SQL for collecting data
  - [ ] Python library
  - [ ] JavaScript / TypeScript library
