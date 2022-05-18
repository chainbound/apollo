# Schema Examples

## Calculate the price and size of every ETH-USDC swap
```hcl
contract usdc_to_eth_swaps "0x905dfCD5649217c42684f23958568e533C711Aa3" {
  abi = "unipair.abi.json"

  // amount0Out = ETH out
  // amount1In = USDC in 
  event Swap {
    outputs = ["amount1In", "amount0Out", "amount0In", "amount1Out"]
  }


  save {
    timestamp = timestamp
    block = blocknumber
    contract = contract_address
    tx_hash = tx_hash

    price = amount0Out != 0 ? (parse_decimals(amount1In, 6) / parse_decimals(amount0Out, 18)) : (parse_decimals(amount1Out, 6) / parse_decimals(amount0In, 18))
    dir = amount0Out != 0 ? "buy" : "sell"
    size = amount1In != 0 ? parse_decimals(amount1In, 6) : parse_decimals(amount1Out, 6)
  }
}
```