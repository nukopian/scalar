# JSONPath Compliance Test Suite

`cts.json` is the [JSONPath Compliance Test Suite](https://github.com/jsonpath-standard/jsonpath-compliance-test-suite)
for RFC 9535, at commit `20ebdcce5008c8a777ee9c80ad853ecf606950b5`, unchanged. It's used under the BSD-2
licence in `LICENSE`, with the copyright notice in `NOTICE`.

`TestCompliance` (in `../../jsonpath_cts_test.go`) runs it against `Find`.
To update it, download `cts.json` from a newer commit and change the
commit above.
