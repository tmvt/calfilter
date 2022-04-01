# Calendar Filter

This tool finds and removes events that match specific filter rules.
It is especially helpful for calendars you cannot modify but that contain information you want to remove.
The parser is rather rudimentary and only works with `VEVENT` entries at the moment.

Calfilter does not use any libraries and is very lightweight, so it can be run on the tiniest devices.

## Usage

To use the calendar filter, you have to clone and build the project.
The resulting binary file can then be deployed to your preference.
[Here](https://go.dev/doc/tutorial/compile-install) you will find some information about building a Go project.

At runtime, the tool starts a tiny HTTP server which listens on the specified port (see *Config*) and responds to request at `/filter`.
In order to not leak your calendar data, the endpoint is "protected" using a key you specify. 
It is passed to the server using the `key` query parameter. 
A query could look like this: 
```shell
curl -X GET -sL 'http://localhost/filter?key=aGVsbG8gdGhlcmUK'
```
This URL can also be embedded into nearly every calendar program.

## Configuration

Currently, the configuration is stored in a JSON file, which is admittedly not very pretty.
Maybe it will be changed to YAML in the future.

The tool expects the config file to be in the same directory and with the name `config.json`.
An example config file can be found [here](config.json.EXAMPLE).
Most variables should be self-explanatory.

The filtering rules are grouped in order to allow more powerful filters.
Inside a group, the individual rules can be combined so that at least one rule has to match in order to remove an event — `"mode": 0`.
They can also be configured to only remove an event, if all rules of the group match — `"mode": 1`.

A rule always has the event attribute as the key and the actual *RegEx* filter as the value:
```json
{"SUMMARY": "\\d"}
```
This example will remove all events that contain a digit inside the title / summary.
**Because this config is stored in JSON, special characters have to be escaped, which is why we need two backslashes here.**
