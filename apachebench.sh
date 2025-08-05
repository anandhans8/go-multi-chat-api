#!/bin/sh
# post_loc.txt contains the json you want to post
# -p means to POST it
# -H adds an Auth header (could be Basic or Token)
# -T sets the Content-Type
# -c is concurrent clients
# -n is the number of requests to run in the test

ab -p post-message.json -T application/json -H 'Authorization: Bearer Token' -c 10 -n 1000 http://localhost:8080/v1/send/message > ab-output.txt