/*
Copyright 2019 Ridecell, Inc.

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

package rdssnapshot_test

import (
	"testing"

	"github.com/onsi/ginkgo"
	"github.com/onsi/gomega"

	"github.com/Ridecell/ridecell-operator/pkg/controller/rdssnapshot"
	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/service/rds"

	. "github.com/onsi/gomega"
)

func TestRDSSnapshot(t *testing.T) {
	gomega.RegisterFailHandler(ginkgo.Fail)
	ginkgo.RunSpecs(t, "rdssnapshot controller Suite @aws @snapshot")
}

var _ = ginkgo.BeforeSuite(func() {
	testHelpers = test_helpers.Start(rdssnapshot.Add, false)
})

var _ = ginkgo.AfterSuite(func() {
	testHelpers.Stop()
	// If the database exists delete it
	if rdsInstanceID != nil {
		_, err := rdssvc.DeleteDBInstance(&rds.DeleteDBInstanceInput{
			DBInstanceIdentifier: rdsInstanceID,
			SkipFinalSnapshot:    aws.Bool(true),
		})
		Expect(err).ToNot(HaveOccurred())
	}
})