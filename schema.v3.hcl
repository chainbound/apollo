chain = "arbitrum"

contract usdc_eth_reserves "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"

  // Call methods
  method getReserves {
    // These are the outputs we're interested in. They are available 
    // to transform as variables in the "save" block below. Outputs should
    // be provided as a list.
    outputs = ["_reserve0", "_reserve1"]

    // The "save" block defines which data we want to save, and how it should look.
    // Basic arithmetic works here, as well as some more advanced functions like parse_decimals.
    // Other than that, the "save" block provides access to variables like "timestamp", "blocknumber", and "contract_address".
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

  // Listen for events
  event Swap {
    // The outputs we're interested in, same way as with methods.
    outputs = ["amount1In", "amount0Out"]

    // Besides the normal context, the "save" block for events provides an additional
    // variable "tx_hash".
    save {
      time = timestamp
      block = blocknumber
      tx = tx_hash 
      eth_in = parse_decimals(amount0In, 18)
      usdc_out = parse_decimals(amount1Out, 6)
      
      // Example: we're interested in the execution price.
      price = parse_decimals(amount1Out, 6) / parse_decimals(amount0In, 18)
    }
  }

  method balanceOf {
    // Inputs should be defined as a map.
    inputs = {
      address = "0xe1Dd30fecAb8a63105F2C035B084BfC6Ca5B1493"
    }

    outputs = ["balance"]

    save {
      // timestamp = timestamp
      block = blocknumber
      contract = contract_address
      account = address
      // account_balance = parse_decimals(balance, 18)
    }
  }
}

