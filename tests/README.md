
### How Travis works?
- For this Project, travis is configured to start a matrix to run two different jobs, each using a different environment
- One environment is using go which is always triggered with each commit, the other one is using python and it is always triggered
  but will never run the tests unless it was triggered using scheduled cron jobs

### Trigger Manual builds

#### To trigger a manual build using travis, use one of the following ways:

1- Using Travis Dashboard
- Go to [travis-beta-features](https://travis-ci.org/features), and enable the the Dashboard option then a click on the Travis CI logo at the top gets you there!
- Once you are there, you can trigger a manual build as shown in figure. This build is triggered from the default branch
![trigger](https://github.com/zero-os/0-core/blob/docs-patch-1/tests/pics/dashboard-repo.png)

2- Using trigger_travis.sh script
- The advantage of this script is that the build can be triggered from any branch. Here is the script [trigger_travis](https://github.com/zero-os/0-core/blob/cron-jobs/tests/trigger_travis.sh)
- For this script to work, a travis token need to be provided. To generate token, you need to install line command travis client [travis-client](https://github.com/travis-ci/travis.rb#installation), then use these commands:
    ```
    travis login --org
    travis token --org
    ```
- For instance, to trigger a build from master branch, the branch "master" and the token should be passed to the script
    ```
    bash trigger_travis.sh master l17-fmjUgycEAcQWWCA
    ```
### Python G8os Tests
- Here is the link for the tests: [testsuite](https://github.com/zero-os/0-core/tree/cron-jobs/tests/testsuite)
- When travis triggers the python environment, Basically it starts to create packet machine using latest g8os image, then run the tests on it.
- After the tests are done, the packet machine is deleted.
