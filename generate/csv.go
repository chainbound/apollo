package generate

func GenerateCsvHeader(cs ContractSchemaV2) []string {
	columns := []string{"timestamp", "blocknumber", "chain", "contract"}

	// The only dynamic table columns are the arguments and the return values
	for _, call := range cs.Methods() {
		for arg := range call.Args() {
			columns = append(columns, arg)
		}

		columns = append(columns, call.Outputs()...)
	}

	return columns
}
