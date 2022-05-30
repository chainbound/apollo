loop {
  items = ["ethereum", "arbitrum"]

  query pairs_created {
    // Each query can have a different chain
    chain = item

    event PairCreated {
      abi = "unipair.abi.json"
      outputs = ["token0", "token1", "pair"]

    }

    // Besides the normal context, the "save" block for events provides an additional
    // variable "tx_hash". "save" blocks are at the query-level and have access to variables
    // defined in the "transform" block
    save {
      timestamp = timestamp
      block = blocknumber
      token0 = token0
      token1 = token1
      pair = pair
    }
  }
}