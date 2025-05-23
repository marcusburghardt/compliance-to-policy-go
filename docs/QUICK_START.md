## C2P Quick Start

Below is a C2P workspace setup and end-to-end workflow with multiple C2P Plugins.

### Usage of C2P CLI
```
C2P CLI

Usage:
  c2pcli [command]

Available Commands:
  completion   Generate the autocompletion script for the specified shell
  help         Help about any command
  oscal2policy Transform OSCAL to policy artifacts.
  result2oscal Transform policy result artifact to OSCAL Assessment Results.
  tools        Tools for working with OSCAL Documents
  version      Display version

Flags:
  -h, --help   help for c2pcli

Use "c2pcli [command] --help" for more information about a command.
```

### Example workflows with multiple PVPs

> Note: For specific usage of `ocm` or `kyverno`, see their respective directories under `docs`.


## Set up the C2P CLI workspace


1. Build the C2P CLI and artifacts
    ```bash
    # move c2pcli into your path from ./bin 
    make build
    make build-plugins
    ```

2. Copy the plugins to your plugin directory.
    ```bash
   # The default is c2p-plugins and that is what is used in the below scripts.
   # You can override this in the CLI with "-p <plugin-dir>"
   bash ./hack/regenerate-manifests.sh
   ```

## Run the C2P CLL

1. Review the `c2p-config`

   ```bash
   cat docs/c2p-config.yaml
   mkdir /tmp/outputs /tmp/kyverno /tmp/ocm
   ```
   Note: When reviewing the OSCAL Component Definition file, note the two `validation` components and their titles. This is how the C2P plugin manager selects the plugins.
   
2. Generate policy artifacts with the `c2pcli`
   ```bash
   c2pcli oscal2policy -c docs/c2p-config.yaml -n nist_800_53
   # All generated manifests should be under the below directory
   # which can be applied via `kublectl`.
   ls /tmp/outputs
   ```
   
   **Note on --name**  
   --name or -n is the short name for the control source for a particular control
   implementation. This helps you select which baseline to run. It could be the
   same short name used with `compliance-trestle` if using the `trestle://` prefix, or it is documented in the
   separate property on the control implementation(example below)
   
   ```json
        {
          "name": "Framework_Short_Name",
          "ns": "https://oscal-compass.github.io/compliance-trestle/schemas/oscal",
          "value": "nist_800_53"
        }
   ```
   
3. Generate an OSCAL Assessment Result with the `c2pcli`
   ```bash
   c2pcli result2oscal -c docs/c2p-config.yaml -n nist_800_53 -o /tmp/assessment-results.json
   cat /tmp/assessment-results.json
   ```
   
4. Generate a compliance posture markdown file with the `c2pcli`
   ```bash
   c2pcli oscal2posture -c ./docs/c2p-config.yaml --assessment-results /tmp/assessment-results.json -o /tmp/compliance-posture.md
   ```