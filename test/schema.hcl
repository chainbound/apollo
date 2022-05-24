
// contract usdc_eth_reserves "0x905dfCD5649217c42684f23958568e533C711Aa3" {
//   abi = "unipair.abi.json"

//   // Call methods
//   method getReserves {
//     // These are the outputs we're interested in. They are available 
//     // to transform as variables in the "save" block below. Outputs should
//     // be provided as a list.
//     outputs = ["_reserve0", "_reserve1"]
//   }

//   save {
//     block = blocknumber
//     timestamp = timestamp
//     eth_reserve = parse_decimals(_reserve0, 18)
//     usdc_reserve = parse_decimals(_reserve1, 6)
//     mid_price = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)
//   }
// }
chain = "arbitrum"

variables = {
  test_string = "hello"
  test_number = 10
}

// A query defines one output (csv file, SQL table, ...)
// Everything in a query needs to be something that's happening at the same time:
// * multiple methods on different contracts
// * multiple methods on the same contracts
// * single events
query usdc_eth_swaps {
  chain = "arbitrum"

  contract usdc_to_eth_swaps "0x905dfCD5649217c42684f23958568e533C711Aa3" {
    abi = "unipair.abi.json"
    // Listen for events
    event Swap {
      // The outputs we're interested in, same way as with methods.
      outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]

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

