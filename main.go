package main

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
// limitations under the License.

// This example uses Device Token authentication to login using Azure AD
// and creates a Resource Group called Quickstart-VM and adds a
// storage acct.  In the storage account it creates a blob container
// and writes a file to the container.  In order to create the file the
// access keys for the storage container are retrieved and used.

import (
	"bufio"
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/services/resources/mgmt/2019-05-01/resources"
	"github.com/Azure/azure-sdk-for-go/services/storage/mgmt/2019-06-01/storage"
	"github.com/Azure/azure-storage-blob-go/azblob"
	"github.com/Azure/go-autorest/autorest"
	"github.com/Azure/go-autorest/autorest/azure/auth"
	"github.com/Azure/go-autorest/autorest/to"
)

const (
	resourceGroupName = "Quickstart-RG"
	location          = "eastus"
)

var (
	ctx        = context.Background()
	numpad     = randomNumString(6)
	authorizer autorest.Authorizer
	appId      *string
	tenant     *string
	subId      *string
)

// Authenticate with the Azure services using file-based authentication
func init() {
	subId = flag.String("subid", "", "Azure SubscriptionId (Required)")
	appId = flag.String("appid", "", "App Registration Id (Required)")
	tenant = flag.String("tenantid", "", "Tenant Id (Required)")
	flag.Parse()
	if *subId == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *appId == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	if *tenant == "" {
		flag.PrintDefaults()
		os.Exit(1)
	}
	dfc := auth.NewDeviceFlowConfig(*appId, *tenant)
	spToken, err := dfc.ServicePrincipalToken()
	authorizer = autorest.NewBearerAuthorizer(spToken)
	if err != nil {
		log.Fatal(err)
	}
}

func main() {
	// Create Resource Group for Storage Accounts
	group, err := createGroup()
	if err != nil {
		log.Fatalf("failed to create group: %v", err)
	}
	log.Printf("Created group: %v", *group.Name)

	// Create storageAcct1
	storageAcct1 := fmt.Sprintf("acct%s", numpad)
	log.Printf("Creating storageAcct1: %s", storageAcct1)
	result1, err := createStorageAcct(storageAcct1)
	if err != nil {
		log.Fatalf("Failed to create storage: %v", err)
	}
	log.Printf("Completed storage creation %v: %v", storageAcct1, result1.ProvisioningState)

	// Get Access Key for storageAcct1
	log.Printf("Getting access key1 for %v", storageAcct1)
	// Grab all the keys from storageAcct1
	store1Keys := getStorageKeys(storageAcct1).Keys
	// Let use the first one only
	key1 := *((*store1Keys)[0]).Value

	// Create container1 in storageAcct1
	container1 := fmt.Sprintf("cont%s", numpad)
	_, err = createStorageContainer(storageAcct1, container1)
	if err != nil {
		log.Fatalf("Failed to create storage container: %v", err)
	} else {
		log.Printf("Completed storage container creation %v ", container1)
	}

	// Create a file in container1 of storageAcct1
	createFileinStorageAcct(storageAcct1, container1, key1)

	// Check to delete resourceGroup
	if confirm("Do you want to delete the Resource Group " + resourceGroupName, 3) {
		deleteGroup()
		log.Println("Resource Group deleted")
	} else {
		log.Println("Leaving Resource Group: " + resourceGroupName)
	}
	log.Println("All Done!  Thanks for playing.")
}

func confirm(s string, tries int) bool {
	r := bufio.NewReader(os.Stdin)
	for ; tries > 0; tries-- {
		fmt.Printf("%s [y/n]: ", s)

		res, err := r.ReadString('\n')
		if err != nil {
			log.Fatal(err)
		}

		// Empty input (i.e. "\n")
		if len(res) < 2 {
			continue
		}
		return strings.ToLower(strings.TrimSpace(res))[0] == 'y'
	}
	return false
}

// Produce a 0 padded random string of a certain size.
// For instance if size=6 a zero padded string will be
// produced between 000000 and 999999
func randomNumString(size int) string {
	var max string
	if size > 10 || size < 2 {
		log.Fatal("Must be greater than 1 or less than 10")
	}
	rand.Seed(time.Now().Unix())
	for i := 1; i < size; i++ {
		max = max + "9"
	}
	maxint, err := strconv.Atoi(max)
	if err != nil {
		log.Fatal("cannot convert to number")
	}
	num := fmt.Sprintf("%0*d", size, rand.Intn(maxint))
	return num
}

// Create a resource group for the deployment.
func createGroup() (group resources.Group, err error) {
	groupsClient := resources.NewGroupsClient(*subId)
	groupsClient.Authorizer = authorizer
	group, err = groupsClient.CreateOrUpdate(
		ctx,
		resourceGroupName,
		resources.Group{
			Location: to.StringPtr(location)})
	if err != nil {
		log.Fatal(err)
	}
	return group, nil
}

// Delete a resource group
func deleteGroup() (resp autorest.Response , err error) {
	groupsClient := resources.NewGroupsClient(*subId)
	groupsClient.Authorizer = authorizer
	groupsFuture, err := groupsClient.Delete(
		ctx,
		resourceGroupName,
	)
	if err != nil {
		log.Fatal(err)
	}

	err = groupsFuture.Future.WaitForCompletionRef(
		ctx,
		groupsClient.BaseClient.Client,
	)
	if err != nil {
		log.Fatal(err)
	}

	return groupsFuture.Result(groupsClient)
}

func getStorageKeys(storageAcct string) (keys storage.AccountListKeysResult) {

	storageClient := storage.NewAccountsClient(*subId)
	storageClient.Authorizer = authorizer

	keys, err := storageClient.ListKeys(
		ctx,
		resourceGroupName,
		storageAcct,
		"kerb",
	)

	if err != nil {
		log.Fatal(err)
	}
	//accountsFuture.Result(storageClient)
	return keys
}

// Create the storageAcct
func createStorageAcct(storageName string) (storageacct storage.Account, err error) {

	storageClient := storage.NewAccountsClient(*subId)
	storageClient.Authorizer = authorizer

	accountsFuture, err := storageClient.Create(
		ctx,
		resourceGroupName,
		storageName,
		storage.AccountCreateParameters{
			Sku: &storage.Sku{
				Name: storage.StandardLRS},
			Location:                          to.StringPtr(location),
			AccountPropertiesCreateParameters: &storage.AccountPropertiesCreateParameters{}},
	)
	if err != nil {
		log.Fatal(err)
	}
	err = accountsFuture.Future.WaitForCompletionRef(
		ctx,
		storageClient.BaseClient.Client,
	)

	if err != nil {
		log.Fatal(err)
	}

	return accountsFuture.Result(storageClient)
}

func createStorageContainer(storageAcct string, containerName string) (container storage.BlobContainer, err error) {
	blobClient := storage.NewBlobContainersClient(*subId)
	blobClient.Authorizer = authorizer

	container, err = blobClient.Create(
		ctx,
		resourceGroupName,
		storageAcct,
		containerName,
		storage.BlobContainer{},
	)
	if err != nil {
		log.Fatal(err)
	}
	return container, nil
}

func createFileinStorageAcct(storageAcct string, container string, accessKey string) {
	// Create a file for upload
	data := []byte("hello from Microsoft. this is a blob " + numpad + "\n")
	fileName := "file-" + numpad
	err := ioutil.WriteFile(fileName, data, 0700)
	if err != nil {
		log.Fatal(err)
	}

	cred, err := azblob.NewSharedKeyCredential(storageAcct, accessKey)
	if err != nil {
		log.Fatal(err)
	}

	pipeline := azblob.NewPipeline(cred, azblob.PipelineOptions{})

	URL, _ := url.Parse(fmt.Sprintf("https://%s.blob.core.windows.net/%s", storageAcct, container))

	containerURL := azblob.NewContainerURL(*URL, pipeline)

	blobURL := containerURL.NewBlockBlobURL(fileName)

	file, err := os.Open(fileName)

	fmt.Printf("Uploading file with blob name: %s\n\n", fileName)

	_, err = azblob.UploadFileToBlockBlob(
		ctx,
		file,
		blobURL,
		azblob.UploadToBlockBlobOptions{
			BlockSize:   4 * 1024 * 1024,
			Parallelism: 16},
	)
	if err != nil {
		log.Fatal(err)
	}

	defer file.Close()
	file.Close()
}
