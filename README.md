# Azure SDK for Go quickstart: Creating a VM

This code compiles out of the box but requires you to have a valid subscription
to Azure and an Azure AD app registration with proper permissions.  See the
aad.sh script

This script will create a ResourceGroup, Storage Acct and Container.  It will 
grab the access keys from the created Storage Acct and then use a key to 
upload a blob to the container.  It will then prompt you to either leave the
resource group or delete it.

# License

// Copyright (c) Microsoft and contributors.  All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
// http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
//
// See the License for the specific language governing permissions and
// limitations under the License
