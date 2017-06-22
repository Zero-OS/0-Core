## How Travis works?
- For this Project, travis is configured to start a matrix to run two different jobs, each using a different environment
- One environment is using go which is always triggered with each commit, the other one is using python and it is always triggered
  but will never run the tests unless it was triggered using scheduled cron jobs

## Trigger Manual builds

- To trigger a manual build using travis, please use this script: [trigger_travis](https://github.com/zero-os/0-core/blob/cron-jobs/tests/trigger_travis.sh)
- For this script to work, a travis token need to be provided. To generate token, you need to install line command travis client [travis-client](https://github.com/travis-ci/travis.rb#installation), then use these commands:
    ```
    travis login --org
    travis token --org
    ```
- For instance, to trigger a build from master branch, the branch "master" and the token should be passed to the script
    ```
    bash trigger_travis.sh master l17-fmjUgycEAcQWWCA
    ```
