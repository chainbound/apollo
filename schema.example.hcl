chain = "arbitrum"

contract usdc_balance "0xFF970A61A04b1cA14834A43f5dE4533eBDDB5CC8" {
  abi = "erc20.abi.json"

  method balanceOf {
    // Inputs should be defined as a map.
    inputs = {
      address = "0xCe2CC46682E9C6D5f174aF598fb4931a9c0bE68e"
    }

    outputs = ["balance"]
  }

  save {
    block = blocknumber
    timestamp = timestamp
    usdc_address = contract_address
    account = address
    account_balance = parse_decimals(balance, 18)
  }
}

