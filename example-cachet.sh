#!/bin/bash

webhookURL="http://127.0.0.1:8080/webhook"
debug=true

function fail() {
	echo "Failed to curl..."
	exit 1
}

rawPayload='
{
  "status": "firing",
  "alerts": [
    {
      "labels": {
   	    "alertname": "High Latency",
	      "service":   "my-service",
	      "severity":  "critical",
	      "instance":  "somewhere"
       },
       "annotations": {
    	    "summary": "The latency is too damn high!"
       }
    }
  ]
}'

if (${debug}); then
	alert=$(echo $rawPayload | jq '.')
	echo $alert
fi

curl -XPOST -d "$rawPayload" "${webhookURL}" || fail

echo -e "\nPress enter to resolve."
read

rawPayload='
{
  "status": "resolved",
  "alerts": [
    {
      "labels": {
   	    "alertname": "High Latency",
	      "service":   "my-service",
	      "severity":  "critical",
	      "instance":  "somewhere"
       },
       "annotations": {
    	    "summary": "The latency is too damn high!"
       }
    }
  ]
}'

if (${debug}); then
        alert=$(echo $rawPayload | jq '.')
        echo $alert
fi

curl -XPOST -d "$rawPayload" "${webhookURL}" || fail
