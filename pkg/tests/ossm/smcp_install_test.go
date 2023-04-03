// Copyright 2021 Red Hat, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ossm

import (
	_ "embed"
	"fmt"
	"testing"

	"github.com/maistra/maistra-test-tool/pkg/util/check/assert"
	"github.com/maistra/maistra-test-tool/pkg/util/env"
	"github.com/maistra/maistra-test-tool/pkg/util/hack"
	"github.com/maistra/maistra-test-tool/pkg/util/oc"
	"github.com/maistra/maistra-test-tool/pkg/util/retry"
	"github.com/maistra/maistra-test-tool/pkg/util/test"
	. "github.com/maistra/maistra-test-tool/pkg/util/test"
)

type vars struct {
	Name      string
	Namespace string
}

func installSMCPVersion(t test.TestHelper, smcpTemplate string, vars interface{}) {
	t.LogStep("Install SMCP")
	oc.ApplyTemplate(t, meshNamespace, smcpTemplate, vars)
	if env.IsRosa() {
		oc.PatchWithMerge(
			t, meshNamespace,
			fmt.Sprintf("smcp/%s", smcpName),
			`{"spec":{"security":{"identity":{"type":"ThirdParty"}}}}`)
	}
	oc.WaitSMCPReady(t, meshNamespace, smcpName)
	oc.ApplyString(t, meshNamespace, smmr)
	t.LogStep("Check SMCP is Ready")
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.WaitCondition(t, meshNamespace, "smcp", smcpName, "Ready")
	})
}

func uninstallSMCPVersion(t test.TestHelper, smcpTemplate string, vars interface{}) {
	t.LogStep("Delete SMCP in namespace " + meshNamespace)
	oc.DeleteFromString(t, meshNamespace, smmr)
	oc.DeleteFromTemplate(t, meshNamespace, smcpTemplate, vars)
	retry.UntilSuccess(t, func(t TestHelper) {
		oc.AllResourcesDeleted(t,
			meshNamespace,
			assert.OutputContains("No resources found in",
				"All resources deleted from namespace",
				"Still waiting for resources to be deleted from namespace"))
	})
}

func TestSMCPInstall(t *testing.T) {
	NewTest(t).Id("A1").Groups(Smoke, Full, InterOp, ARM).Run(func(t TestHelper) {
		hack.DisableLogrusForThisTest(t)
		t.Cleanup(func() {
			oc.DeleteNamespace(t, meshNamespace)
			SetupNamespacesAndControlPlane()
		})
		vars := vars{
			Name:      smcpName,
			Namespace: meshNamespace,
		}
		versionTemplates := map[string]string{
			"2.1": smcpV21_template,
			"2.2": smcpV22_template,
			"2.3": smcpV23_template,
			// "2.4": smcpV24_template,
		}
		smcpVersion := env.GetDefaultSMCPVersion()
		_, ok := versionTemplates[smcpVersion]
		if !ok {
			t.Errorf("Unsupported SMCP version: %s", smcpVersion)
			return
		}
		for version, smcpTemplate := range versionTemplates {
			t.NewSubTest("smcp_test_install_" + version).Run(func(t TestHelper) {
				t.LogStep("Create Namespace and Install SMCP v" + version)
				installSMCPVersion(t, smcpTemplate, vars)
				uninstallSMCPVersion(t, smcpTemplate, vars)
			})
		}

		t.NewSubTest("smcp_test_upgrade_2.1_to_2.2").Run(func(t TestHelper) {
			installSMCPVersion(t, "2.1", vars)
			t.LogStep("Upgrade SMCP from v2.1 to v2.2")
			installSMCPVersion(t, "2.2", vars)
		})

		t.NewSubTest("smcp_test_upgrade_2.2_to_2.3").Run(func(t TestHelper) {
			installSMCPVersion(t, "2.2", vars)
			t.LogStep("Upgrade SMCP from v2.2 to v2.3")
			installSMCPVersion(t, "2.3", vars)
		})

		t.NewSubTest("smcp_test_upgrade_2.3_to_2.4").Run(func(t TestHelper) {
			installSMCPVersion(t, "2.3", vars)
			t.LogStep("Upgrade SMCP from v2.3 to v2.4")
			installSMCPVersion(t, "2.4", vars)
		})
	})
}
