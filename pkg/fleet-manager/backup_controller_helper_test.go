/*
Copyright Kurator Authors.
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

package fleet

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

func TestBuildVeleroBackupInstance(t *testing.T) {
	cases := []struct {
		name        string
		description string
		creatorName string
		// velero backup can be created by kurator backup or migrate
		creatorKind  string
		creatorLabel string
		clusterName  string
		backupSpec   backupapi.BackupSpec
	}{
		{
			name: "include-ns",
			description: "Test the scenario where the backup includes specific namespaces " +
				"and the Velero backup instance is created by Kurator 'Backup' with the creator name 'include-ns'.",
			creatorName:  "include-ns",
			creatorKind:  BackupKind,
			creatorLabel: BackupNameLabel,
			clusterName:  "kurator-member1",
			backupSpec: backupapi.BackupSpec{
				Destination: backupapi.Destination{
					Fleet: "quickstart",
					Clusters: []*corev1.ObjectReference{
						{
							Kind: "AttachedCluster",
							Name: "kurator-member1",
						},
					},
				},
				Policy: &backupapi.BackupPolicy{
					ResourceFilter: &backupapi.ResourceFilter{
						IncludedNamespaces: []string{
							"kurator-backup",
						},
					},
					TTL: metav1.Duration{Duration: time.Hour * 24 * 30},
				},
			},
		},
		{
			name: "label-selector",
			description: "Test the case where the backup is filtered based on label selectors, " +
				"and the Velero backup instance is created by Kurator 'Migrate' with the creator name 'label-selector'.",
			creatorName:  "label-selector",
			creatorKind:  MigrateKind,
			creatorLabel: MigrateNameLabel,
			clusterName:  "kurator-member2",
			backupSpec: backupapi.BackupSpec{
				Destination: backupapi.Destination{
					Fleet: "quickstart",
					Clusters: []*corev1.ObjectReference{
						{
							Kind: "AttachedCluster",
							Name: "kurator-member2",
						},
					},
				},
				Policy: &backupapi.BackupPolicy{
					ResourceFilter: &backupapi.ResourceFilter{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"app": "busybox2",
							},
						},
					},
					TTL: metav1.Duration{Duration: time.Hour * 24 * 10},
				},
			},
		},
	}

	typeMeta := &metav1.TypeMeta{
		APIVersion: "velero.io/v1",
		Kind:       "Backup",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// get expect backup yaml
			expectedYAML, err := getExpectedBackup(tc.name)
			assert.NoError(t, err)

			backupLabels := generateVeleroInstanceLabel(tc.creatorLabel, tc.creatorName, tc.backupSpec.Destination.Fleet)
			veleroBackupName := generateVeleroResourceName(tc.clusterName, tc.creatorKind, tc.creatorName)

			// get actual backup yaml
			actualBackup := buildVeleroBackupInstanceForTest(&tc.backupSpec, backupLabels, veleroBackupName, typeMeta)
			actualYAML, err := yaml.Marshal(actualBackup)
			if err != nil {
				t.Fatalf("failed to marshal actual output to YAML: %v", err)
			}

			assert.Equal(t, string(expectedYAML), string(actualYAML))
		})
	}
}

func TestBuildVeleroScheduleInstance(t *testing.T) {
	cases := []struct {
		name        string
		description string
		creatorName string
		// velero backup can be created by kurator backup
		creatorKind  string
		creatorLabel string
		clusterName  string
		backupSpec   *backupapi.BackupSpec
	}{
		{
			name: "schedule",
			description: "Test the scenario where a backup schedule is set to '0 0 * * *' (daily). " +
				"The Velero schedule instance is created by Kurator 'Backup' with the creator name 'include-ns' targeting the 'kurator-member1' cluster.",
			creatorName:  "schedule",
			creatorKind:  BackupKind,
			creatorLabel: BackupNameLabel,
			clusterName:  "kurator-member1",
			backupSpec: &backupapi.BackupSpec{
				Schedule: "0 0 * * *",
				Destination: backupapi.Destination{
					Fleet: "quickstart",
					Clusters: []*corev1.ObjectReference{
						{
							Kind: "AttachedCluster",
							Name: "kurator-member1",
						},
					},
				},
				Policy: &backupapi.BackupPolicy{
					ResourceFilter: &backupapi.ResourceFilter{
						IncludedNamespaces: []string{
							"kurator-backup",
						},
					},
					TTL: metav1.Duration{Duration: time.Hour * 24 * 30},
				},
			},
		},
	}

	typeMeta := &metav1.TypeMeta{
		APIVersion: "velero.io/v1",
		Kind:       "Schedule",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// get expect schedule yaml
			expectedYAML, err := getExpectedBackup(tc.name)
			assert.NoError(t, err)

			scheduleLabels := generateVeleroInstanceLabel(tc.creatorLabel, tc.creatorName, tc.backupSpec.Destination.Fleet)
			scheduleName := generateVeleroResourceName(tc.clusterName, tc.creatorKind, tc.creatorName)

			// get actual schedule yaml
			actualSchedule := buildVeleroScheduleInstanceForTest(tc.backupSpec, scheduleLabels, scheduleName, typeMeta)
			actualYAML, err := yaml.Marshal(actualSchedule)
			if err != nil {
				t.Fatalf("failed to marshal actual output to YAML: %v", err)
			}

			assert.Equal(t, string(expectedYAML), string(actualYAML))
		})
	}
}

func getExpectedBackup(caseName string) ([]byte, error) {
	return os.ReadFile("testdata/backup/backup/" + caseName + ".yaml")
}

func TestBuildVeleroRestoreInstance(t *testing.T) {
	cases := []struct {
		name        string
		description string
		creatorName string
		// velero backup can be created by kurator restore or migrate
		creatorKind      string
		creatorLabel     string
		clusterName      string
		veleroBackupName string
		restoreSpec      *backupapi.RestoreSpec
	}{
		{
			name: "default",
			description: "Test the default restore scenario where the Velero restore instance is created by Kurator 'Restore' with the creator name 'include-ns'. " +
				"The restore targets the 'kurator-member1' cluster using the backup named 'include-ns'.",
			creatorName:      "default",
			creatorKind:      RestoreKind,
			creatorLabel:     RestoreNameLabel,
			clusterName:      "kurator-member1",
			veleroBackupName: "kurator-member1-backup-include-ns",
			restoreSpec: &backupapi.RestoreSpec{
				BackupName: "include-ns",
				Destination: &backupapi.Destination{
					Fleet: "quickstart",
					Clusters: []*corev1.ObjectReference{
						{
							Kind: "AttachedCluster",
							Name: "kurator-member1",
						},
					},
				},
			},
		},
		{
			name: "custom-policy",
			description: "Test the custom policy restore scenario where resources are filtered based on the 'env: test' label. " +
				"The Velero restore instance is created by Kurator 'Migrate' with the creator name 'include-ns', targeting the 'kurator-member1' cluster.",
			creatorName:      "custom-policy",
			creatorKind:      MigrateKind,
			creatorLabel:     MigrateNameLabel,
			clusterName:      "kurator-member1",
			veleroBackupName: "kurator-member1-migrate-include-ns",
			restoreSpec: &backupapi.RestoreSpec{
				BackupName: "include-ns",
				Destination: &backupapi.Destination{
					Fleet: "quickstart",
					Clusters: []*corev1.ObjectReference{
						{
							Kind: "AttachedCluster",
							Name: "kurator-member1",
						},
					},
				},
				Policy: &backupapi.RestorePolicy{
					ResourceFilter: &backupapi.ResourceFilter{
						LabelSelector: &metav1.LabelSelector{
							MatchLabels: map[string]string{
								"env": "test",
							},
						},
					},
				},
			},
		},
	}

	typeMeta := &metav1.TypeMeta{
		APIVersion: "velero.io/v1",
		Kind:       "Restore",
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// get expect restore yaml
			expectedYAML, err := getExpectedRestore(tc.name)
			assert.NoError(t, err)

			// just for test, the real fleet name may not record in `tc.restoreSpec.Destination.Fleet`
			restoreLabels := generateVeleroInstanceLabel(tc.creatorLabel, tc.creatorName, tc.restoreSpec.Destination.Fleet)
			restoreName := generateVeleroResourceName(tc.clusterName, tc.creatorKind, tc.creatorName)

			// get actual restore yaml
			actualBackup := buildVeleroRestoreInstanceForTest(tc.restoreSpec, restoreLabels, tc.veleroBackupName, restoreName, typeMeta)
			actualYAML, err := yaml.Marshal(actualBackup)
			if err != nil {
				t.Fatalf("failed to marshal actual output to YAML: %v", err)
			}

			assert.Equal(t, string(expectedYAML), string(actualYAML))
		})
	}
}

func getExpectedRestore(caseName string) ([]byte, error) {
	return os.ReadFile("testdata/backup/restore/" + caseName + ".yaml")
}
