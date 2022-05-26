start_time = format_date("02-01-2006 15:04", "25-05-2022 12:00")
end_time = now

variables = {
  b = upper("eth_buy")
  s = upper("eth_sell")
}

// query defines the name of your query -> name of your output files and SQL tables
query usdc_eth_swaps {
  // Each query can have a different chain
  chain = "arbitrum"

  contract "0x905dfCD5649217c42684f23958568e533C711Aa3" {
    abi = "unipair.abi.json"
    // Listen for events
    event Swap {
      // The outputs we're interested in, same way as with methods.
      outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
    }

    // "transform" blocks are at the contract-level
    transform {
      usdc_sold = parse_decimals(amount1In, 6)
      eth_sold = parse_decimals(amount0In, 18)

      usdc_bought = parse_decimals(amount1Out, 6)
      eth_bought = parse_decimals(amount0Out, 18)

      buy = amount0Out != 0
    }
  }

  filter = [
    eth_bought != 0
  ]

  // Besides the normal context, the "save" block for events provides an additional
  // variable "tx_hash". "save" blocks are at the query-level and have access to variables
  // defined in the "transform" block
  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    // Example: we want to calculate the price of the swap.
    // We have to make sure we don't divide by 0, so we use the ternary operator.
    swap_price = eth_bought != 0 ? (usdc_sold / eth_bought) : (usdc_bought / eth_sold)
    direction = buy ? b : s
    size_in_udsc = eth_bought != 0 ? usdc_sold : usdc_bought
  }
}

