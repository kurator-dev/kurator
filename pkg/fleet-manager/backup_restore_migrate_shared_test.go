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
	velerov1 "github.com/vmware-tanzu/velero/pkg/apis/velero/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/yaml"

	backupapi "kurator.dev/kurator/pkg/apis/backups/v1alpha1"
)

const backupTestDataPath = "backup-testdata/backup/"

// buildVeleroBackupInstanceForTest is a helper function for testing for buildVeleroBackupInstance, which constructs a Velero Backup instance with a specified TypeMeta.
func buildVeleroBackupInstanceForTest(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string, typeMeta *metav1.TypeMeta) *velerov1.Backup {
	veleroBackup := buildVeleroBackupInstance(backupSpec, labels, veleroBackupName)
	veleroBackup.TypeMeta = *typeMeta // set TypeMeta for test
	return veleroBackup
}

// buildVeleroScheduleInstanceForTest is a helper function for testing buildVeleroScheduleInstance, which constructs a Velero Schedule instance with a specified TypeMeta.
func buildVeleroScheduleInstanceForTest(backupSpec *backupapi.BackupSpec, labels map[string]string, veleroBackupName string, typeMeta *metav1.TypeMeta) *velerov1.Schedule {
	veleroSchedule := buildVeleroScheduleInstance(backupSpec, labels, veleroBackupName)
	veleroSchedule.TypeMeta = *typeMeta
	return veleroSchedule
}

func TestBuildVeleroBackupInstance(t *testing.T) {
	cases := []struct {
		name        string
		description string
		creatorName string
		// velero backup can be created by kurator backup or migrate
		creatorKind      string
		creatorLabel     string
		clusterName      string
		creatorNamespace string
		backupSpec       backupapi.BackupSpec
	}{
		{
			name: "include-ns",
			description: "Test the scenario where the backup includes specific namespaces " +
				"and the Velero backup instance is created by Kurator 'Backup' with the creator name 'include-ns'.",
			creatorName:      "include-ns",
			creatorNamespace: "default",
			creatorKind:      BackupKind,
			creatorLabel:     BackupNameLabel,
			clusterName:      "kurator-member1",
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
			creatorName:      "label-selector",
			creatorNamespace: "default",
			creatorKind:      MigrateKind,
			creatorLabel:     MigrateNameLabel,
			clusterName:      "kurator-member2",
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
			veleroBackupName := generateVeleroResourceName(tc.clusterName, tc.creatorKind, tc.creatorNamespace, tc.creatorName)

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
		creatorKind      string
		creatorLabel     string
		clusterName      string
		creatorNamespace string
		backupSpec       *backupapi.BackupSpec
	}{
		{
			name: "schedule",
			description: "Test the scenario where a backup schedule is set to '0 0 * * *' (daily). " +
				"The Velero schedule instance is created by Kurator 'Backup' with the creator name 'include-ns' targeting the 'kurator-member1' cluster.",
			creatorName:      "schedule",
			creatorNamespace: "default",
			creatorKind:      BackupKind,
			creatorLabel:     BackupNameLabel,
			clusterName:      "kurator-member1",
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
			scheduleName := generateVeleroResourceName(tc.clusterName, tc.creatorKind, tc.creatorNamespace, tc.creatorName)

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
	return os.ReadFile(backupTestDataPath + caseName + ".yaml")
}

func TestAllBackupsCompleted(t *testing.T) {
	tests := []struct {
		name     string
		status   backupapi.BackupStatus
		expected bool
	}{
		{
			name: "No details",
			status: backupapi.BackupStatus{
				Details: nil,
			},
			expected: true,
		},
		{
			name: "Backup not completed",
			status: backupapi.BackupStatus{
				Details: []*backupapi.BackupDetails{
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseInProgress,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Backup completed",
			status: backupapi.BackupStatus{
				Details: []*backupapi.BackupDetails{
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseCompleted,
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Multiple backups, one not completed",
			status: backupapi.BackupStatus{
				Details: []*backupapi.BackupDetails{
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseCompleted,
						},
					},
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseInProgress,
						},
					},
				},
			},
			expected: false,
		},
		{
			name: "Multiple backups, all completed",
			status: backupapi.BackupStatus{
				Details: []*backupapi.BackupDetails{
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseCompleted,
						},
					},
					{
						BackupStatusInCluster: &velerov1.BackupStatus{
							Phase: velerov1.BackupPhaseCompleted,
						},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := allBackupsCompleted(tt.status)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestMostRecentCompletedBackup(t *testing.T) {
	time1 := metav1.NewTime(time.Now())
	time2 := metav1.NewTime(time.Now().Add(-10 * time.Minute))
	time3 := metav1.NewTime(time.Now().Add(-20 * time.Minute))

	tests := []struct {
		name     string
		backups  []velerov1.Backup
		expected velerov1.Backup
	}{
		{
			name:     "No backups",
			backups:  []velerov1.Backup{},
			expected: velerov1.Backup{},
		},
		{
			name: "All backups in progress",
			backups: []velerov1.Backup{
				{
					Status: velerov1.BackupStatus{
						Phase:          velerov1.BackupPhaseInProgress,
						StartTimestamp: &time1,
					},
				},
			},
			expected: velerov1.Backup{},
		},
		{
			name: "Single backup completed",
			backups: []velerov1.Backup{
				{
					Status: velerov1.BackupStatus{
						Phase:          velerov1.BackupPhaseCompleted,
						StartTimestamp: &time1,
					},
				},
			},
			expected: velerov1.Backup{
				Status: velerov1.BackupStatus{
					Phase:          velerov1.BackupPhaseCompleted,
					StartTimestamp: &time1,
				},
			},
		},
		{
			name: "Multiple backups, mixed phases",
			backups: []velerov1.Backup{
				{
					Status: velerov1.BackupStatus{
						Phase:          velerov1.BackupPhaseInProgress,
						StartTimestamp: &time1,
					},
				},
				{
					Status: velerov1.BackupStatus{
						Phase:          velerov1.BackupPhaseCompleted,
						StartTimestamp: &time2,
					},
				},
				{
					Status: velerov1.BackupStatus{
						Phase:          velerov1.BackupPhaseCompleted,
						StartTimestamp: &time3,
					},
				},
			},
			expected: velerov1.Backup{
				Status: velerov1.BackupStatus{
					Phase:          velerov1.BackupPhaseCompleted,
					StartTimestamp: &time2,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MostRecentCompletedBackup(tt.backups)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetCronInterval(t *testing.T) {
	tests := []struct {
		name      string
		cronExpr  string
		expected  time.Duration
		expectErr bool
	}{
		{
			name:      "Invalid cron expression",
			cronExpr:  "invalid",
			expectErr: true,
		},
		{
			name:      "Every minute",
			cronExpr:  "* * * * *",
			expected:  time.Minute + 30*time.Second,
			expectErr: false,
		},
		{
			name:      "Every 5 minutes",
			cronExpr:  "*/5 * * * *",
			expected:  5*time.Minute + 30*time.Second,
			expectErr: false,
		},
		{
			name:      "Every hour",
			cronExpr:  "0 * * * *",
			expected:  time.Hour + 30*time.Second,
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			interval, err := GetCronInterval(tt.cronExpr)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expected, interval)
			}
		})
	}
}
