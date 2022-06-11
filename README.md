# Apollo
> Program for easily querying and collecting EVM chaindata based on a schema.

![apollo-diagram drawio](./docs/apollo-flow.png)

## Documentation
For detailed documentation, visit [apollo.chainbound.io](https://apollo.chainbound.io).

## Installation
```bash
go install github.com/chainbound/apollo
```

## Usage
### Setting up
First, generate the config directory and files:
```
apollo init
```
This will generate the configuration files (`config.yml` and `schema.hcl`) and put it into your configuration
directory, which will either be `$XDG_CONFIG_HOME/apollo` or `$HOME/.config/apollo`. This is the directory
in which you have to configure `apollo`, and it's also the directory where `apollo` will try to find the specified
contract ABIs.

### Running
* **Realtime mode**

After defining the schema (with `time_interval`), run
```bash
apollo --realtime --stdout
```

* **Historical mode**

After defining the schema with `start`, `end` and `interval` parameters, just run
```bash
apollo --stdout
```


## Output
There are 3 output options:
* `stdout`: this will just print the results to your terminal.
* `csv`: this will save your output into a csv file. The name of your file will be the name of your `query`. The other columns
will be made up of what's defined in the `save` block.
* `db`: this will save your output into a Postgres SQL table, with the table name matching your `query` name. The settings are defined in `config.yml` in your `apollo` config directory.
