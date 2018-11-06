/*
Copyright 2018 Ridecell, Inc..

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

package statefulset

import (
	postgresv1 "github.com/zalando-incubator/postgres-operator/pkg/apis/acid.zalan.do/v1"
	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

type statefulsetComponent struct {
	templatePath    string
	waitForDatabase bool
}

func New(templatePath string, waitForDatabase bool) *statefulsetComponent {
	return &statefulsetComponent{templatePath: templatePath, waitForDatabase: waitForDatabase}
}

func (comp *statefulsetComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&appsv1.StatefulSet{},
	}
}

func (comp *statefulsetComponent) IsReconcilable(ctx *components.ComponentContext) bool {
	// If we need the database, make sure that exists. Otherwise, always ready.
	if comp.waitForDatabase {
		instance := ctx.Top.(*summonv1beta1.SummonPlatform)
		return instance.Status.PostgresStatus != nil && *instance.Status.PostgresStatus == postgresv1.ClusterStatusRunning && instance.Spec.Version == instance.Status.MigrateVersion
	} else {
		return true
	}
}

func (comp *statefulsetComponent) Reconcile(ctx *components.ComponentContext) (reconcile.Result, error) {
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, func(goalObj, existingObj runtime.Object) error {
		goal := goalObj.(*appsv1.StatefulSet)
		existing := existingObj.(*appsv1.StatefulSet)
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	return res, err
}