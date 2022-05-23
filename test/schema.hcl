chain = "arbitrum"

contract usdc_eth_reserves_test "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"

  // Call methods
  // method getReserves {
  //   // These are the outputs we're interested in. They are available 
  //   // to transform as variables in the "save" block below. Outputs should
  //   // be provided as a list.
  //   outputs = ["_reserve0", "_reserve1"]
  // }


  // Listen for events
  event Swap {
    // The outputs we're interested in, same way as with methods.
    outputs = ["amount1In", "amount0Out"]

  // Call methods
    method getReserves {
      // These are the outputs we're interested in. They are available 
      // to transform as variables in the "save" block below. Outputs should
      // be provided as a list.
      outputs = ["_reserve0", "_reserve1"]
    }

  }

  // The "save" block defines which data we want to save, and how it should look.
  // Basic arithmetic works here, as well as some more advanced functions like parse_decimals.
  // Other than that, the "save" block provides access to variables like "timestamp", "blocknumber", and "contract_address".
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx = tx_hash
    eth_reserve = parse_decimals(_reserve0, 18)
    usdc_reserve = parse_decimals(_reserve1, 6)
    eth_in = parse_decimals(amount0In, 18)
    usdc_out = parse_decimals(amount1Out, 6)

    // Example: we want to calculate the mid price from the 2 reserves.
    // Cannot reuse variables that are defined in the same "save" block.
    // We have to reuse variables that were defined in advance, i.e.
    // in "inputs" or "outputs"
    mid_price = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)
  }
}

