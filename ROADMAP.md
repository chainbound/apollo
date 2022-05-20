# Roadmap

- [ ] **v1.0.1-beta**
  - [x] Logging with log levels
  - [x] Improved stdout output
  - [ ] Timestamps for setting start, end and interval options
  - [ ] Review concurrency model + the fact that logs are always overfetched. 
  - [x] Working DB output
  - [ ] Ability to call methods when logs occur, and not just at a random interval. This would make it easier for some use cases. The DSL syntax will be defining a method block inside of an event block
  - [ ] Filter lists
      - Example: don't save certain log outputs if one of the values is 0.  
  - [ ] Standalone logs (not emitted from a certain contract)
  - [ ] Should be able to get basic Ethereum balances too
  - [ ] transform blocks, which are contract level blocks to define and transform variables to be used later in the top-level save blocks 
  - [ ] Save block should be a top-level block so that we can do cross-contract operations
      - This only works for results that happen at the same time, i.e. method calls

- [ ] **v1.1.0-beta**
  - [ ] Transaction monitoring
      - You would be able to filter historical transactions based on certain predicates: value thresholds, sender and receiver addresses, gas prices and amounts, or certain method calls or inputs.
  - [ ] Mempool monitoring
      - You would be able to monitor mempool transactions and save them based on a predicate. Same as above. 
  - [ ] Different stream output option for latency-sensitive operations (like mempool monitoring): i.e. Websocket, SSE 
      - Latency sensitive operations would probably also need different evaluation options. I think evaluating everything in the save block might take some time, would need to benchmark that. An option is to just not have a save block and stream everything as-is, let the application take care of decoding.
  - [ ] JSON output
  - [ ] Generalized SQL output (MySQL, SQL Server)
  - [ ] Aggregation operations like group by, sum, avg
  - [ ] Unverified methods and events

- [ ] **v1.2.0-beta**
  - [ ] Transaction simulation at certain times / events