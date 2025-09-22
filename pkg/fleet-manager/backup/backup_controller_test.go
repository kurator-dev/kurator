/*
Copyright 2022-2025 Kurator Authors.
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
	http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package backup

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client/fake"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
	fleetapi "kurator.dev/kurator/pkg/apis/fleet/v1alpha1"
)

const (
	testFleetName  = "test-fleet"
	testNamespace  = "default"
	testBackupName = "test-backup"
)

func setupTest(t *testing.T) *BackupManager {
	scheme := runtime.NewScheme()

	if err := backupapi.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add backupapi to scheme: %v", err)
	}
	if err := fleetapi.AddToScheme(scheme); err != nil {
		t.Fatalf("Failed to add fleetapi to scheme: %v", err)
	}

	client := fake.NewClientBuilder().WithScheme(scheme).Build()

	mgr := &BackupManager{Client: client, Scheme: scheme}

	return mgr
}

// createTestReconcileRequest creates a test Reconcile request for the given Backup object.
func createTestReconcileRequest(backup *backupapi.Backup) reconcile.Request {
	if backup == nil {
		return reconcile.Request{}
	}
	return reconcile.Request{
		NamespacedName: types.NamespacedName{
			Name:      backup.Name,
			Namespace: backup.Namespace,
		},
	}
}

// createTestBackup creates a test Backup for the given Backup name and namespace.
func createTestBackup(name, namespace string) *backupapi.Backup {
	return &backupapi.Backup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: backupapi.BackupSpec{
			Destination: backupapi.Destination{
				Fleet: testFleetName,
			},
		},
	}
}

func createTestFleet(name, namespace string) *fleetapi.Fleet {
	return &fleetapi.Fleet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
	}
}

func TestReconcile(t *testing.T) {
	tests := []struct {
		name       string
		backup     *backupapi.Backup
		wantResult ctrl.Result
		wantErr    bool
	}{
		{
			name:       "Backup without finalizer",
			backup:     createTestBackup(testBackupName, testNamespace),
			wantResult: ctrl.Result{},
			wantErr:    false,
		},
		{
			name: "Backup with deletion timestamp",
			backup: func() *backupapi.Backup {
				b := createTestBackup(testBackupName, testNamespace)
				now := metav1.Now()
				b.DeletionTimestamp = &now
				return b
			}(),
			wantResult: ctrl.Result{},
			wantErr:    false,
		},
		{
			name:       "Normal backup",
			backup:     createTestBackup(testBackupName, testNamespace),
			wantResult: ctrl.Result{},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := setupTest(t)
			fleetObj := createTestFleet(testFleetName, testNamespace)
			if err := mgr.Client.Create(context.Background(), fleetObj); err != nil {
				t.Fatalf("Failed to create test fleet: %v", err)
			}

			if err := mgr.Client.Create(context.Background(), tt.backup); err != nil {
				t.Fatalf("Failed to create test backup: %v", err)
			}

			ctx := context.TODO()
			req := createTestReconcileRequest(tt.backup)

			gotResult, gotErr := mgr.Reconcile(ctx, req)
			assert.Equal(t, tt.wantResult, gotResult)
			if tt.wantErr {
				assert.NotNil(t, gotErr)
			} else {
				assert.Nil(t, gotErr)
			}
		})
	}
}

func TestReconcileBackupResources(t *testing.T) {
	tests := []struct {
		name    string
		backup  *backupapi.Backup
		wantErr bool
	}{
		{
			name: "Test scheduled backup",
			backup: func() *backupapi.Backup {
				b := createTestBackup(testBackupName, testNamespace)
				b.Spec.Schedule = "test-schedule"
				return b
			}(),
			wantErr: false,
		},
		{
			name:    "Test one-time backup",
			backup:  createTestBackup(testBackupName, testNamespace),
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := setupTest(t)
			fleetObj := createTestFleet(testFleetName, testNamespace)
			if err := mgr.Client.Create(context.Background(), fleetObj); err != nil {
				t.Fatalf("Failed to create test fleet: %v", err)
			}

			if err := mgr.Client.Create(context.Background(), tt.backup); err != nil {
				t.Fatalf("Failed to create test backup: %v", err)
			}

			_, gotErr := mgr.reconcileBackupResources(context.TODO(), tt.backup, nil)

			if tt.wantErr {
				assert.NotNil(t, gotErr)
			} else {
				assert.Nil(t, gotErr)
			}
		})
	}
}

func TestReconcileDeleteBackup(t *testing.T) {
	tests := []struct {
		name          string
		backup        *backupapi.Backup
		wantErr       bool
		wantFinalizer bool
	}{
		{
			name: "Successful deletion",
			backup: func() *backupapi.Backup {
				b := createTestBackup(testBackupName, testNamespace)
				controllerutil.AddFinalizer(b, BackupFinalizer)
				return b
			}(),
			wantErr:       false,
			wantFinalizer: false,
		},
		{
			name: "Failed deletion due to fetch error",
			backup: func() *backupapi.Backup {
				b := createTestBackup("non-existent", "non-existent")
				controllerutil.AddFinalizer(b, BackupFinalizer)
				return b
			}(),
			wantErr:       true,
			wantFinalizer: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr := setupTest(t)
			fleetObj := createTestFleet(testFleetName, testNamespace)
			if err := mgr.Client.Create(context.Background(), fleetObj); err != nil {
				t.Fatalf("Failed to create test fleet: %v", err)
			}

			if err := mgr.Client.Create(context.Background(), tt.backup); err != nil {
				t.Fatalf("Failed to create test backup: %v", err)
			}

			_, err := mgr.reconcileDeleteBackup(context.TODO(), tt.backup)

			if tt.wantErr {
				assert.NotNil(t, err)
			} else {
				assert.Nil(t, err)
			}

			if tt.wantFinalizer {
				assert.Contains(t, tt.backup.Finalizers, BackupFinalizer)
			} else {
				assert.NotContains(t, tt.backup.Finalizers, BackupFinalizer)
			}
		})
	}
}
