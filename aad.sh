#!/bin/bash 

# Find IDs for Microsoft services that require permissions
# az ad sp list --filter "displayName eq 'Azure Storage'" -o json --all 
# az ad sp list --filter "displayName eq 'Microsoft Graph'" -o json --all 
# az ad sp list --filter "displayName eq 'Windows Azure Service Management API'" -o json --all 

command -v az 2&> /dev/null
if [ $? -ne 0 ]; then
    echo "ERROR: Requires Azure CLI (az).  Aborting..."
    exit 1
fi

command -v jq 2&> /dev/null
if [ $? -ne 0 ]; then
    echo "ERROR: Requires JQuery (jq). Aborting..."
    exit 1
fi

RND=$(echo $RANDOM | grep -o ...$)
GRAPH_ID=00000003-0000-0000-c000-000000000000
STORAGE_ID=e406a681-f3d4-42a8-90b6-c2b029497af1
COMPUTE_ID=797f4846-ba00-4fd7-ba43-dac1f8f63013
OPEN_ID=$(az ad sp show --id $GRAPH_ID --query "oauth2Permissions[?value=='openid'].id | [0]" -otsv)
PROFILE_ID=$(az ad sp show --id $GRAPH_ID --query "oauth2Permissions[?value=='profile'].id | [0]" -otsv)
USER_READ_ID=$(az ad sp show --id $GRAPH_ID --query "oauth2Permissions[?value=='User.Read'].id | [0]" -otsv)
SUSERIM_ID=$(az ad sp show --id $STORAGE_ID --query "oauth2Permissions[?value=='user_impersonation'].id | [0]" -otsv)
CUSERIM_ID=$(az ad sp show --id $COMPUTE_ID --query "oauth2Permissions[?value=='user_impersonation'].id | [0]" -otsv)
REDIRECT_URL="https://login.microsoftonline.com/common/oauth2/nativeclient"
DISPLAY_NAME=app-aad-$RND

echo "Microsoft Graph ID:           $GRAPH_ID"
echo "Openid:                       $OPEN_ID"
echo "Profile:                      $PROFILE_ID"
echo "User.Read:                    $USER_READ_ID"
echo "Azure Storage ID:             $STORAGE_ID"
echo "storage user_impersonation:   $SUSERIM_ID"
echo "compute user impersonation:   $CUSERIM_ID"
echo -e "\nDISPLAY_NAME:                $DISPLAY_NAME\n" 

JSON=$(cat <<-EOF
[{
  "resourceAppId": "$COMPUTE_ID",
  "resourceAccess": [
    {
      "id": "${CUSERIM_ID}",
      "type": "Scope"
    }
  ] 
},
{
  "resourceAppId": "$STORAGE_ID",
  "resourceAccess": [
    {
      "id": "${SUSERIM_ID}",
      "type": "Scope"
    }
  ] 
 },
 {
  "resourceAppId": "${GRAPH_ID}",
  "resourceAccess": [
    {
      "id": "${OPEN_ID}",
      "type": "Scope"
    },
    {
      "id": "${PROFILE_ID}",
      "type": "Scope"
    },
    {
      "id": "${USER_READ_ID}",
      "type": "Scope"
    }
  ]
}]
EOF
)

SM_JSON=$(echo $JSON | jq -c)
echo -e "$SM_JSON\n"

# Verify if we want to proceed
read -p "Are you sure you want continue creating application registration [y/N]?"
if [[ ! "$REPLY" =~ ^[Yy]$ ]]; then
    exit
fi


APP_REG=$(az ad app create \
  --display-name ${DISPLAY_NAME} \
  --password ThisSecretPassw0rd! \
  --reply-urls $REDIRECT_URL \
  --required-resource-accesses $SM_JSON \
  --available-to-other-tenants false \
  --native-app true \
  -o json
)

sleep 30 # Give time for things to propagate

APP_ID=$(echo $APP_REG | jq -r '.appId')
echo "Created Appplication Registration... APPID=$APP_ID"

echo "Creating Service Principal"
SP=$(az ad sp create --id ${APP_ID} -o json)
echo $SP | jq

TENANT_ID=$(echo $SP | jq -r '.appOwnerTenantId')

SUB_ID=$(az account show -o json --query id -o tsv)

sleep 60  # Give time for things to propagate
echo "Creating role assignment for Storage Blob Data Contributor"
az role assignment create \
  --assignee ${APP_ID} \
  --role "Storage Blob Data Contributor" \
  --subscription ${SUB_ID} \
  -o json | jq

echo "APPID:            $APP_ID"
echo "TENANTID:         $TENANT_ID"
echo "SUBSCRIPTIONID:   $SUB_ID"
echo "DISPLAY NAME:     $DISPLAY_NAME"
