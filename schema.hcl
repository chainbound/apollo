chain = "arbitrum"

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
//     timestamp = timestamp
//     block = blocknumber
//     contract = contract_address
//     eth_reserve = parse_decimals(_reserve0, 18)
//     usdc_reserve = parse_decimals(_reserve1, 6)

//     // Example: we want to calculate the mid price from the 2 reserves.
//     // Cannot reuse variables that are defined in the same "save" block.
//     // We have to reuse variables that were defined in advance, i.e.
//     // in "inputs" or "outputs"
//     mid_price = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)

//   }
// }

// contract usdc_balance "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8" {
//   abi = "erc20.abi.json"

//   method balanceOf {
//     // Inputs should be defined as a map.
//     inputs = {
//       address = "0xe1Dd30fecAb8a63105F2C035B084BfC6Ca5B1493"
//     }

//     outputs = ["balance"]
//   }

//   save {
//     account = address
//     account_balance = parse_decimals(balance, 18)
//   }
// }

contract usdc_to_eth_swaps "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"
  // Listen for events
  event Swap {
    // The outputs we're interested in, same way as with methods.
    outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
  }


  // Besides the normal context, the "save" block for events provides an additional
  // variable "tx_hash".
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    // Example: we want to calculate the price of the swap.
    // We have to make sure we don't divide by 0, so we use the ternary operator.
    price = amount0Out != 0 ? (parse_decimals(amount1In, 6) / parse_decimals(amount0Out, 18)) : (parse_decimals(amount1Out, 6) / parse_decimals(amount0In, 18))
    dir = amount0Out != 0 ? "buy" : "sell"
    size = amount1In != 0 ? parse_decimals(amount1In, 6) : parse_decimals(amount1Out, 6)
  }
}

