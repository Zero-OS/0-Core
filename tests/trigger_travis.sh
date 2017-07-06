#!/bin/bash

BRANCH=$1
token=$2
body='{
"request": {
"branch": "'${BRANCH}'",
"config":{
"matrix":{"include":
{
 "language": "python",
  "group": "stable",
  "dist": "precise",
  "after_script": [
    "bash tests/run_tests.sh after"
  ],
  "before_script": [
    "bash tests/run_tests.sh before"
  ],
  "python": 3.5,
  "script": [
    "bash tests/run_tests.sh run"
  ],
  "os": "linux"
}}
}
}}'

curl -s -X POST \
 -H "Content-Type: application/json" \
 -H "Accept: application/json" \
 -H "Travis-API-Version: 3" \
 -H "Authorization: token ${token}" \
 -d "$body" \
 https://api.travis-ci.org/repo/zero-os%2F0-core/requests
