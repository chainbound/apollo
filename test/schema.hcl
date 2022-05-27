variables = {
  arbi = "arbitrum"
  abi = "unipair.abi.json"

  pool = "0x905dfCD5649217c42684f23958568e533C711Aa3"

  dude = 14
}

query usdc_eth_swaps {
  chain = arbi

  contract "pool" {
    abi = abi

    // Listen for events
    event Swap {
      // The outputs we're interested in, same way as with methods.
      outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
      // outputs = [for s in test : upper(s)]

      method getReserves {
        // Call at the previous block
        block_offset = -1
        // These are the outputs we're interested in. They are available 
        // to transform as variables in the "save" block below. Outputs should
        // be provided as a list.
        outputs = ["_reserve0", "_reserve1"]
      }
    }

    transform {
      eth_reserve = parse_decimals(_reserve0, 18)
      usdc_reserve = parse_decimals(_reserve1, 6)
    }
  }

  filter = [
    eth_reserve > 90
  ]

  // Besides the normal context, the "save" block for events provides an additional
  // variable "tx_hash".
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    mid_price = usdc_reserve / eth_reserve

    // Example: we want to calculate the price of the swap.
    // We have to make sure we don't divide by 0, so we use the ternary operator.
    price = amount0Out != 0 ? (parse_decimals(amount1In, 6) / parse_decimals(amount0Out, 18)) : (parse_decimals(amount1Out, 6) / parse_decimals(amount0In, 18))
    dir = amount0Out != 0 ? upper("buy") : "sell"
    size = amount1In != 0 ? parse_decimals(amount1In, 6) : parse_decimals(amount1Out, 6)
  }
}

query v2_pair_created_p2 {
  // Each query can have a different chain
  chain = "ethereum"

  event PairCreated {
    abi = abi
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