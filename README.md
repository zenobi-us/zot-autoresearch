# <EXTENSION_NAME>

> <ONE-SENTENCE_DESCRIPTION_OF_WHAT_THIS_ZOT_EXTENSION_DOES>

<!--
Template checklist:
- Replace every <PLACEHOLDER> in this file.
- Replace the example commands, URLs, and screenshots.
- Remove sections that do not apply to this extension.
- Update extension.json, go.mod, and the Go package name to match the new project.
-->

[![CI](<CI_BADGE_URL>)](<CI_WORKFLOW_URL>)
[![Release](<RELEASE_BADGE_URL>)](<RELEASE_WORKFLOW_URL>)

`<EXTENSION_NAME>` is a [zot](https://github.com/patriceckhart/zot) extension written in Go. It provides <PRIMARY_CAPABILITY> for <TARGET_USERS_OR_WORKFLOW>.

## Features

- <FEATURE_1>
- <FEATURE_2>
- <FEATURE_3>

## Requirements

- [zot](https://github.com/patriceckhart/zot) `<MINIMUM_ZOT_VERSION_OR_COMPATIBILITY_NOTE>`
- Go `<MINIMUM_GO_VERSION>` for local development
- `<OTHER_RUNTIME_OR_SYSTEM_REQUIREMENT>`

## Install

### Install a published release

Download the archive for your platform from [Releases](<RELEASES_URL>) and place the extension where zot can discover it.

```sh
<INSTALL_COMMAND_OR_PLATFORM_SPECIFIC_STEPS>
```

### Build from source

```sh
git clone <REPOSITORY_URL>
cd <REPOSITORY_DIRECTORY>
go build -o <EXTENSION_BINARY_NAME> .
```

Run the extension with zot:

```sh
zot --ext /path/to/<EXTENSION_BINARY_NAME>
```

> **Development note:** The template manifest currently uses `go run main.go`. Replace this with the release/runtime command appropriate for the finished extension before publishing.

## Quick start

1. <STEP_1>
2. <STEP_2>
3. <STEP_3>

Example:

```sh
<QUICK_START_COMMAND>
```

Expected result:

```text
<EXPECTED_OUTPUT_OR_BEHAVIOUR>
```

## Configuration

Describe where configuration is stored, how it is discovered, and which values are required.

```json
{
  "<CONFIGURATION_KEY>": "<CONFIGURATION_VALUE>"
}
```

| Option | Required | Default | Description |
| --- | --- | --- | --- |
| `<OPTION>` | yes/no | `<DEFAULT>` | `<WHAT_IT_CONTROLS>` |

See [`<CONFIGURATION_REFERENCE_FILE>`](<CONFIGURATION_REFERENCE_LINK>) for the complete reference.

## Usage

### <COMMON_USE_CASE>

```text
<COMMAND_OR_ZOT_WORKFLOW>
```

Explain what the user should see and how to verify that the extension is active.

### <SECOND_USE_CASE>

<SHORT, PRACTICAL_EXAMPLE>

## Development

Run the formatter, tests, and build locally:

```sh
gofmt -w .
go test ./...
go build ./...
```

Run the end-to-end tests, if applicable:

```sh
<E2E_TEST_COMMAND>
```

To add a feature:

1. <DEVELOPMENT_STEP_1>
2. <DEVELOPMENT_STEP_2>
3. Add or update tests.
4. Update this README and any configuration reference.

## Extension protocol

Briefly document the protocol messages, events, commands, or SDK interfaces this extension uses. Link to the relevant zot documentation or source:

- Manifest: [`extension.json`](extension.json)
- Entry point: [`main.go`](main.go)
- zot extension documentation: <ZOT_EXTENSION_DOCUMENTATION_URL>

## Project layout

- [`extension.json`](extension.json): extension manifest.
- [`main.go`](main.go): Go entry point and extension implementation.
- [`go.mod`](go.mod): Go module and dependencies.
- `<TEST_DIRECTORY>`: tests and test fixtures.
- `<ADDITIONAL_DOCUMENTATION_FILE>`: <WHAT_IT_DOCUMENTS>.

## Troubleshooting

### The extension does not start

Check that zot can execute `<EXTENSION_BINARY_NAME_OR_COMMAND>` and that the manifest is valid. Run:

```sh
<DIAGNOSTIC_COMMAND>
```

### <COMMON_PROBLEM>

<DIAGNOSIS_AND_FIX>

For additional help, open an issue at [<ISSUE_TRACKER_URL>](<ISSUE_TRACKER_URL>) and include the zot version, operating system, extension version, and relevant logs. Do not include secrets.

## Security

This extension <READS_OR_EXECUTES_OR_TRANSMITS_WHAT>. Treat configuration and downloaded extension binaries as code. Review configuration before enabling the extension, especially in untrusted repositories.

## Contributing

Contributions are welcome. Please read [`<CONTRIBUTING_FILE>`](<CONTRIBUTING_LINK>) before opening a pull request.

Use [Conventional Commits](<CONVENTIONAL_COMMITS_URL>) if this repository's release automation requires them.

## License

<LICENSE_NAME>. See [`LICENSE`](LICENSE) for the full text.

## Status

`<EXTENSION_NAME>` is `<EXPERIMENTAL/BETA/STABLE>`. `<SHORT_STATUS_NOTE_OR_LINK_TO_ROADMAP>`.
