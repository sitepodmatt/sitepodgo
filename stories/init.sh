#!/bin/bash

# use https://github.com/bronze1man/yaml2json and not to shower of shit of yaml/json convert that dont conform in nodejs space

URL="http://localhost:9080"

curlit() {
  curl "$@"
}

post() {
  curlit -s -X POST -d @$1 -H "Content-Type: application/json" "${URL}${2}"
}

echo "Creating sitepod"
SITEPOD_OUTPUT=$(post <(cat new-sitepod.yaml | yaml2json) "/apis/stable.sitepod.io/v1/namespaces/default/sitepods")
echo $SITEPOD_OUTPUT



SITEPOD_UID=$(echo -n "$SITEPOD_OUTPUT" | jq .metadata.uid)
echo -e "$SITEPOD_UID"


#echo "Creating systemuser-matt"
#post <(yaml3json new-system-user.yaml) "/apis/stable.sitepod.io/v1/namespaces/default/systemusers"

# generate a new host key

tempfile=$(mktemp)
echo -e  'y\n'| ssh-keygen -t rsa -f "$tempfile" -N ''
(cat new-ssh-service.yaml | yaml2json) > new-ssh-service.json
#$ splice this into 
jsontempfile=$(mktemp)
{ jq ".spec.sshHostKey = \"$(cat $tempfile)\" | .metadata.labels.sitepod = $SITEPOD_UID" new-ssh-service.json > "$jsontempfile" ; } && mv "$jsontempfile" new-ssh-service.json

echo "Create user ssh service"
post new-ssh-service.json "/apis/stable.sitepod.io/v1/namespaces/default/appcomponents"

echo "Create nginx webserver service"
(cat new-nginx-service.yaml | yaml2json) > new-nginx-service.json
jsontempfile=$(mktemp)
{ jq ".metadata.labels.sitepod = $SITEPOD_UID" new-nginx-service.json > "$jsontempfile" ; } && mv "$jsontempfile" new-nginx-service.json

post new-nginx-service.json "/apis/stable.sitepod.io/v1/namespaces/default/appcomponents"

#cat new-ssh-service.json
#exit


(cat new-system-user.yaml | yaml2json) > new-system-user.json
jsontempfile=$(mktemp)
{ jq ".spec.sshHostKey = \"$(cat $tempfile)\" | .metadata.labels.sitepod = $SITEPOD_UID" new-system-user.json > "$jsontempfile" ; } && mv "$jsontempfile" new-system-user.json

echo "Creating systemuser-matt"
post new-system-user.json "/apis/stable.sitepod.io/v1/namespaces/default/systemusers"

echo "Create new website"
jsontempfile=$(mktemp)
(cat new-website.yaml | yaml2json) > new-website.json
{ jq ".metadata.labels.sitepod = $SITEPOD_UID" new-website.json > "$jsontempfile" ; } && mv "$jsontempfile" new-website.json
post new-website.json "/apis/stable.sitepod.io/v1/namespaces/default/websites"

