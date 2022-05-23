chain = "arbitrum"

start_block = 10
end_block = 20

// Format your dates
start_time = format_date("01-05-2022 12:00:00", "dd-MM-yyyy hh:mm:ss")
end_time = format_date("30-05-2022 22:00:00", "dd-MM-yyyy hh:mm:ss")

// Variable block
variables = {
  eth = "0xeth"
  usdc = "0xusdc"
}


// GENERAL IDEAS =========================================================
// Gets the native asset balance
balance "0xe1Dd30fecAb8a63105F2C035B084BfC6Ca5B1493" {}

// CONTRACT IDEAS =========================================================
contract usdc_eth_reserves "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"

  // Call methods
  method getReserves {
    outputs = ["_reserve0", "_reserve1"]
  }

  transform {
    eth_reserve_usdc = parse_decimals(_reserve0, 18)
    usdc_reserve = parse_decimals(_reserve1, 6)

    mid_price_usdc = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)
  }
}

contract usdt_eth_reserves "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"

  // Call methods
  method getReserves {
    outputs = ["_reserve0", "_reserve1"]
  }

  transform {
    eth_reserve_usdt = parse_decimals(_reserve0, 18)
    usdt_reserve = parse_decimals(_reserve1, 6)

    mid_price_usdt = parse_decimals(_reserve1, 6) / parse_decimals(_reserve0, 18)
  }
}

// Standalone save block to combine outputs from different contract calls
save {
  timestamp = timestamp
  block = blocknumber
  contract = contract_address

  // Might be useful for arbs? This price difference should theoretically be very small.
  price_diff = mid_price_usdc - mid_price_usdt
}

// EVENT IDEAS ===========================================================
// Standalone events
event Transfer {
  abi = "erc20.abi.json"

  outputs = [
    "from",
    "to",
    "value"
  ]

  // Because it could be any ERC20 transfer, we don't
  // know the decimals in advance and need to call them.
  // The code would somehow need to know that it should call this on
  // the contract that emitted the event.
  method decimals {
    outputs = ["decimals"]
  }

  // filter list
  filter = [
    value != 0,
    from == "0x..."
  ]

  save {
    sender = from
    receiver = to
    amount = format_decimals(value, decimals)
  }
}

contract usdc_to_eth_swaps "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"
  // Listen for events
  event Swap {
    // The outputs we're interested in, same way as with methods.
    outputs = ["amount1In", "amount0Out"]
    // Besides the normal context, the "save" block for events provides an additional
    // variable "tx_hash".


    // TODO: something like this? To call a certain method every time an event happens
    method getReserves {
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
    tx_hash = tx_hash

    // Example: we want to calculate the price of the swap.
    // We have to make sure we don't divide by 0, so we use the ternary operator.
    price = amount0Out != 0 ? (parse_decimals(amount1In, 6) / parse_decimals(amount0Out, 18)) : 0
    amount_eth = parse_decimals(amount0Out, 18)
    amount_usdc = parse_decimals(amount1In, 6)
  }
}

