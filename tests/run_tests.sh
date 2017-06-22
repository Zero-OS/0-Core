#!/bin/bash
point=$1
if [ "$TRAVIS_EVENT_TYPE" == "cron" ]
 then
   if [ "$point" == "before" ]
    then
      echo "Before running tests"
      pip3 install -r tests/requirements.txt
      pip3 install client/py-client/.
      pip3 install git+https://github.com/gigforks/packet-python.git
      cd tests; python3 packet_script.py create $PACKET_TOKEN $TRAVIS_BRANCH $TRAVIS_PULL_REQUEST_BRANCH
   elif [ "$point" == "run" ]
    then
      echo "Running tests .."
      cd tests; nosetests -v -s testsuite/a_basic/tests_02_machines.py:Machinetests.test002_create_list_delete_containers
   elif [ "$point" == "after" ]
    then
      cd tests; python3 packet_script.py delete $PACKET_TOKEN
   fi
 else
   echo "Not a cron job" 
fi

