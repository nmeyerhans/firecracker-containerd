// Copyright 2018-2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License"). You may
// not use this file except in compliance with the License. A copy of the
// License is located at
//
//	http://aws.amazon.com/apache2.0/
//
// or in the "license" file accompanying this file. This file is distributed
// on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either
// express or implied. See the License for the specific language governing
// permissions and limitations under the License.

package devmapper

import (
	"context"
	"io/ioutil"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var (
	testCtx = context.Background()
)

func TestPoolMetadata_AddDevice(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	expected := &DeviceInfo{
		Name:       "test2",
		ParentName: "test1",
		Size:       1,
		State:      Activated,
	}

	err := store.AddDevice(testCtx, expected)
	assert.NoError(t, err)

	result, err := store.GetDevice(testCtx, "test2")
	assert.NoError(t, err)

	assert.Equal(t, expected.Name, result.Name)
	assert.Equal(t, expected.ParentName, result.ParentName)
	assert.Equal(t, expected.Size, result.Size)
	assert.Equal(t, expected.State, result.State)
	assert.NotZero(t, result.DeviceID)
	assert.Equal(t, expected.DeviceID, result.DeviceID)
}

func TestPoolMetadata_AddDeviceRollback(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	err := store.AddDevice(testCtx, &DeviceInfo{Name: ""})
	assert.Error(t, err)

	_, err = store.GetDevice(testCtx, "")
	assert.Equal(t, ErrNotFound, err)
}

func TestPoolMetadata_AddDeviceDuplicate(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	err := store.AddDevice(testCtx, &DeviceInfo{Name: "test"})
	assert.NoError(t, err)

	err = store.AddDevice(testCtx, &DeviceInfo{Name: "test"})
	assert.Equal(t, ErrAlreadyExists, err)
}

func TestPoolMetadata_ReuseDeviceID(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	info1 := &DeviceInfo{Name: "test1"}
	err := store.AddDevice(testCtx, info1)
	assert.NoError(t, err)

	info2 := &DeviceInfo{Name: "test2"}
	err = store.AddDevice(testCtx, info2)
	assert.NoError(t, err)

	assert.NotEqual(t, info1.DeviceID, info2.DeviceID)
	assert.NotZero(t, info1.DeviceID)

	err = store.RemoveDevice(testCtx, info2.Name)
	assert.NoError(t, err)

	info3 := &DeviceInfo{Name: "test3"}
	err = store.AddDevice(testCtx, info3)
	assert.NoError(t, err)

	assert.Equal(t, info2.DeviceID, info3.DeviceID)
}

func TestPoolMetadata_RemoveDevice(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	err := store.AddDevice(testCtx, &DeviceInfo{Name: "test"})
	assert.NoError(t, err)

	err = store.RemoveDevice(testCtx, "test")
	assert.NoError(t, err)

	_, err = store.GetDevice(testCtx, "test")
	assert.Equal(t, ErrNotFound, err)
}

func TestPoolMetadata_UpdateDevice(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	oldInfo := &DeviceInfo{
		Name:       "test1",
		ParentName: "test2",
		Size:       3,
		State:      Activated,
	}

	err := store.AddDevice(testCtx, oldInfo)
	assert.NoError(t, err)

	err = store.UpdateDevice(testCtx, oldInfo.Name, func(info *DeviceInfo) error {
		info.ParentName = "test5"
		info.Size = 6
		info.State = Created
		return nil
	})

	assert.NoError(t, err)

	newInfo, err := store.GetDevice(testCtx, "test1")
	require.NoError(t, err)

	assert.Equal(t, "test1", newInfo.Name)
	assert.Equal(t, "test5", newInfo.ParentName)
	assert.EqualValues(t, 6, newInfo.Size)
	assert.Equal(t, Created, newInfo.State)
}

func TestPoolMetadata_GetDeviceNames(t *testing.T) {
	tempDir, store := createStore(t)
	defer cleanupStore(t, tempDir, store)

	err := store.AddDevice(testCtx, &DeviceInfo{Name: "test1"})
	assert.NoError(t, err)

	err = store.AddDevice(testCtx, &DeviceInfo{Name: "test2"})
	assert.NoError(t, err)

	names, err := store.GetDeviceNames(testCtx)
	assert.NoError(t, err)
	require.Len(t, names, 2)

	assert.Equal(t, "test1", names[0])
	assert.Equal(t, "test2", names[1])
}

func createStore(t *testing.T) (tempDir string, store *PoolMetadata) {
	tempDir, err := ioutil.TempDir("", "pool-metadata-")
	require.NoErrorf(t, err, "couldn't create temp directory for metadata tests")

	path := filepath.Join(tempDir, "test.db")
	metadata, err := NewPoolMetadata(path)
	require.NoError(t, err)

	return tempDir, metadata
}

func cleanupStore(t *testing.T, tempDir string, store *PoolMetadata) {
	err := store.Close()
	assert.NoErrorf(t, err, "failed to close metadata store")

	err = os.RemoveAll(tempDir)
	assert.NoErrorf(t, err, "failed to cleanup temp directory")
}
