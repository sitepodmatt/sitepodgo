#!/bin/bash

URL="http://localhost:8080"

curlit() {
  curl "$@"
}

post() {
  curlit -s -X POST -d @$1 -H "Content-Type: application/json" "${URL}${2}"
}

echo "Creating sitepod"
SITEPOD_OUTPUT=$(post <(yaml2json new-sitepod.yaml) "/apis/stable.sitepod.io/v1/namespaces/default/sitepods")
echo $SITEPOD_OUTPUT
exit


SITEPOD_UID=$(echo -n "$SITEPOD_OUTPUT" | jq .metadata.uid)
echo -e "$SITEPOD_UID"

#echo "Creating systemuser-matt"
#post <(yaml3json new-system-user.yaml) "/apis/stable.sitepod.io/v1/namespaces/default/systemusers"

# generate a new host key

tempfile=$(mktemp)
echo -e  'y\n'| ssh-keygen -t rsa -f "$tempfile" -N ''
yaml2json new-ssh-service.yaml > new-ssh-service.json
#$ splice this into 
jsontempfile=$(mktemp)
{ jq ".spec.sshHostKey = \"$(cat $tempfile)\" | .metadata.labels.sitepod = $SITEPOD_UID" new-ssh-service.json > "$jsontempfile" ; } && mv "$jsontempfile" new-ssh-service.json

echo "Create user ssh service"
post new-ssh-service.json "/apis/stable.sitepod.io/v1/namespaces/default/serviceinstances"

#cat new-ssh-service.json
#exit


yaml2json new-system-user.yaml > new-system-user.json
jsontempfile=$(mktemp)
{ jq ".spec.sshHostKey = \"$(cat $tempfile)\" | .metadata.labels.sitepod = $SITEPOD_UID" new-system-user.json > "$jsontempfile" ; } && mv "$jsontempfile" new-system-user.json

echo "Creating systemuser-matt"
post new-system-user.json "/apis/stable.sitepod.io/v1/namespaces/default/systemusers"

#echo "Create user ssh service"
#post <(yaml2json new-ssh-service.yaml) "/apis/stable.sitepod.io/v1/namespaces/default/serviceinstances"

