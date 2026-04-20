#!/bin/sh

RADIUS_HOST="radius-controlplane"
RADIUS_PORT="1813"
SECRET="testing123"

echo "Sending Accounting-Start Request"
echo "--------------------------"
cat /tmp/acct_start.txt | radclient -x "${RADIUS_HOST}:${RADIUS_PORT}" acct "${SECRET}"

sleep 1

echo
echo "Sending Accounting-Stop Response"
echo "--------------------------"
cat /tmp/acct_stop.txt | radclient -x "${RADIUS_HOST}:${RADIUS_PORT}" acct "${SECRET}"

echo
echo "Done."
