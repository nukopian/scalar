# USGS earthquake feed sample

`all_day.json` is the US Geological Survey's "all earthquakes, past day"
GeoJSON feed, as served on 18 October 2026 from
<https://earthquake.usgs.gov/earthquakes/feed/v1.0/summary/all_day.geojson>,
cut to its first 12 quakes with `jq`. `TestReadJSONTakesFeeds` checks that
scalar's JSON reader handles real feed data itself, without handing it to
`encoding/json`.

USGS-authored data is in the public domain:
<https://www.usgs.gov/information-policies-and-instructions/copyrights-and-credits>.
