# secat

`secat` prints `.env` and other key-value files like `cat`, but masks most of each value.
It is meant for AI coding agents: they can see which keys exist, whether a value is set, its rough shape (e.g. `sk-...`) and its length, without the secret itself entering their context.

```console
$ cat .env
OPENAI_API_KEY=sk-proj-AbCdEf1234567890xyzABCD
DB_PASSWORD=hunter2
PORT=3000
EMPTY=

$ secat .env
OPENAI_API_KEY=sk-p***************************
DB_PASSWORD=h******
PORT=3***
EMPTY=
```

The file layout is kept as is: comments, blank lines, indentation, sections, quotes and trailing commas pass through, and only values change.

## Install

```sh
go install github.com/yashikota/secat@latest
```

Prebuilt binaries are available on the [Releases](https://github.com/yashikota/secat/releases) page.

## Usage

```
secat [flags] [file ...]
```

Files are printed one after another. With no file, or with `-`, `secat` reads stdin.

| Flag | Default | Meaning |
|---|---|---|
| `--ratio` | `0.25` | Fraction of each value shown from its head (0 to 1) |
| `--max` | `4` | Maximum number of visible characters per value |
| `--version` | | Print the version |

Each value shows `min(floor(length × ratio), max)` characters (counted in Unicode characters), and the rest become `*`.
The output keeps the original length.

If a file cannot be read, `secat` reports it on stderr, keeps going with the remaining files, and exits with status 1.
Invalid flags exit with status 2.

## Supported formats

`secat` reads each line on its own and masks anything that looks like a key and a value, whatever the file type.

| Line | Typical files |
|---|---|
| `KEY=value`, `export KEY=value` | `.env` |
| `key = value` | INI, `~/.aws/credentials`, TOML, properties |
| `key: value` | YAML, properties |
| `"key": "value",` | pretty-printed JSON |
| `- value`, `"value",` | YAML and JSON list items |

Values that span lines are masked in full:

- quoted values that do not close on their line, including TOML `"""` and `'''`
- YAML block scalars (`key: |`, `key: >-`, ...)
- PEM blocks from `-----BEGIN` to `-----END`
- unquoted values continued with a trailing `\`

## Limitations

- Minified (single-line) JSON is not masked.
- A YAML plain scalar that continues onto the next line without quotes or a block indicator is masked only on its first line.
- Plain items of a YAML flow sequence spread over several lines (`hosts: [` followed by `  a.example.com,`) are not masked.
- INI continuation lines (an indented line continuing the previous value) are not masked.
- Whitespace-separated properties (`key value`) are not recognized.
- Text right after a closing quote (`KEY="abc"def`) is printed as is.
- Values are masked whatever the key is, so non-secret values such as `PORT` are masked too.
