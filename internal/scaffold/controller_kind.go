// Copyright 2018 The Operator-SDK Authors
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

package scaffold

import (
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"unicode"

	"github.com/operator-framework/operator-sdk/internal/scaffold/input"
)

// ControllerKind is the input needed to generate a pkg/controller/<kind>/<kind>_controller.go file
type ControllerKind struct {
	input.Input

	// Resource defines the inputs for the controller's primary resource
	Resource *Resource
	// CustomImport holds the import path for a built-in or custom Kubernetes
	// API that this controller reconciles, if specified by the scaffold invoker.
	CustomImport string

	// The following fields will be overwritten by GetInput().
	//
	// ImportMap maps all imports destined for the scaffold to their import
	// identifier, if any.
	ImportMap map[string]string
	// GoImportIdent is the import identifier for the API reconciled by this
	// controller.
	GoImportIdent string
}

func (s *ControllerKind) GetInput() (input.Input, error) {
	if s.Path == "" {
		fileName := s.Resource.LowerKind + "_controller.go"
		s.Path = filepath.Join(ControllerDir, s.Resource.LowerKind, s.Resource.Version, fileName)
	}
	// Error if this file exists.
	s.IfExistsAction = input.Error
	s.TemplateBody = controllerKindTemplate

	// Set imports.
	if err := s.setImports(); err != nil {
		return input.Input{}, err
	}
	return s.Input, nil
}

func (s *ControllerKind) setImports() (err error) {
	s.ImportMap = controllerKindImports
	importPath := ""
	if s.CustomImport != "" {
		importPath, s.GoImportIdent, err = getCustomAPIImportPathAndIdent(s.CustomImport)
		if err != nil {
			return err
		}
	} else {
		importPath = path.Join(s.Repo, "pkg", "apis", s.Resource.GoImportGroup, s.Resource.Version)
		s.GoImportIdent = s.Resource.GoImportGroup + s.Resource.Version
	}
	// Import identifiers must be unique within a file.
	for p, id := range s.ImportMap {
		if s.GoImportIdent == id && importPath != p {
			// Append "api" to the conflicting import identifier.
			s.GoImportIdent = s.GoImportIdent + "api"
			break
		}
	}
	s.ImportMap[importPath] = s.GoImportIdent
	return nil
}

func getCustomAPIImportPathAndIdent(m string) (p string, id string, err error) {
	sm := strings.Split(m, "=")
	for i, e := range sm {
		if i == 0 {
			p = strings.TrimSpace(e)
		} else if i == 1 {
			id = strings.TrimSpace(e)
		}
	}
	if p == "" {
		return "", "", fmt.Errorf(`custom import "%s" path is empty`, m)
	}
	if id == "" {
		if len(sm) == 2 {
			return "", "", fmt.Errorf(`custom import "%s" identifier is empty, remove "=" from passed string`, m)
		}
		sp := strings.Split(p, "/")
		if len(sp) > 1 {
			id = sp[len(sp)-2] + sp[len(sp)-1]
		} else {
			id = sp[0]
		}
		id = strings.ToLower(id)
	}
	idb := &strings.Builder{}
	// By definition, all package identifiers must be comprised of "_", unicode
	// digits, and/or letters.
	for _, r := range id {
		if unicode.IsDigit(r) || unicode.IsLetter(r) || r == '_' {
			if _, err := idb.WriteRune(r); err != nil {
				return "", "", err
			}
		}
	}
	return p, idb.String(), nil
}

var controllerKindImports = map[string]string{
	"k8s.io/api/core/v1":                                           "corev1",
	"k8s.io/apimachinery/pkg/api/errors":                           "",
	"k8s.io/apimachinery/pkg/apis/meta/v1":                         "metav1",
	"k8s.io/apimachinery/pkg/runtime":                              "",
	"k8s.io/apimachinery/pkg/types":                                "",
	"sigs.k8s.io/controller-runtime/pkg/client":                    "",
	"sigs.k8s.io/controller-runtime/pkg/controller":                "",
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil": "",
	"sigs.k8s.io/controller-runtime/pkg/handler":                   "",
	"sigs.k8s.io/controller-runtime/pkg/manager":                   "",
	"sigs.k8s.io/controller-runtime/pkg/reconcile":                 "",
	"sigs.k8s.io/controller-runtime/pkg/log":                       "logf",
	"sigs.k8s.io/controller-runtime/pkg/source":                    "",
}

const controllerKindTemplate = `package {{ .Resource.Version }}

import (
	"context"

	{{range $p, $i := .ImportMap -}}
	{{$i}} "{{$p}}"
	{{end}}
)

var log = logf.Log.WithName("controller_{{ .Resource.LowerKind }}_{{ .Resource.Version }}")

/**
* USER ACTION REQUIRED: This is a scaffold file intended for the user to modify with their own Controller
* business logic.  Delete these comments after modifying this file.*
 */

// Add creates a new {{ .Resource.Kind }} Controller and adds it to the Manager. The Manager will set fields on the Controller
// and Start it when the Manager is Started.
func Add(mgr manager.Manager) error {
	return add(mgr, newReconciler(mgr))
}

// newReconciler returns a new reconcile.Reconciler
func newReconciler(mgr manager.Manager) reconcile.Reconciler {
	return &Reconcile{{ .Resource.Kind }}{client: mgr.GetClient(), scheme: mgr.GetScheme()}
}

// add adds a new Controller to mgr with r as the reconcile.Reconciler
func add(mgr manager.Manager, r reconcile.Reconciler) error {
	// Create a new controller
	c, err := controller.New("{{ .Resource.LowerKind }}-{{ .Resource.Version }}-controller", mgr, controller.Options{Reconciler: r})
	if err != nil {
		return err
	}

	// Watch for changes to primary resource {{ .Resource.Kind }}
	err = c.Watch(&source.Kind{Type: &{{ .GoImportIdent }}.{{ .Resource.Kind }}{}}, &handler.EnqueueRequestForObject{})
	if err != nil {
		return err
	}

	// TODO(user): Modify this to be the types you create that are owned by the primary resource
	// Watch for changes to secondary resource Pods and requeue the owner {{ .Resource.Kind }}
	err = c.Watch(&source.Kind{Type: &corev1.Pod{}}, &handler.EnqueueRequestForOwner{
		IsController: true,
		OwnerType:    &{{ .GoImportIdent }}.{{ .Resource.Kind }}{},
	})
	if err != nil {
		return err
	}

	return nil
}

// blank assignment to verify that Reconcile{{ .Resource.Kind }} implements reconcile.Reconciler
var _ reconcile.Reconciler = &Reconcile{{ .Resource.Kind }}{}

// Reconcile{{ .Resource.Kind }} reconciles a {{ .Resource.Kind }} object
type Reconcile{{ .Resource.Kind }} struct {
	// This client, initialized using mgr.Client() above, is a split client
	// that reads objects from the cache and writes to the apiserver
	client client.Client
	scheme *runtime.Scheme
}

// Reconcile reads that state of the cluster for a {{ .Resource.Kind }} object and makes changes based on the state read
// and what is in the {{ .Resource.Kind }}.Spec
// TODO(user): Modify this Reconcile function to implement your Controller logic.  This example creates
// a Pod as an example
// Note:
// The Controller will requeue the Request to be processed again if the returned error is non-nil or
// Result.Requeue is true, otherwise upon completion it will remove the work from the queue.
func (r *Reconcile{{ .Resource.Kind }}) Reconcile(request reconcile.Request) (reconcile.Result, error) {
	reqLogger := log.WithValues("Request.Namespace", request.Namespace, "Request.Name", request.Name)
	reqLogger.Info("Reconciling {{ .Resource.Kind }}")

	// Fetch the {{ .Resource.Kind }} instance
	instance := &{{ .GoImportIdent }}.{{ .Resource.Kind }}{}
	err := r.client.Get(context.TODO(), request.NamespacedName, instance)
	if err != nil {
		if errors.IsNotFound(err) {
			// Request object not found, could have been deleted after reconcile request.
			// Owned objects are automatically garbage collected. For additional cleanup logic use finalizers.
			// Return and don't requeue
			return reconcile.Result{}, nil
		}
		// Error reading the object - requeue the request.
		return reconcile.Result{}, err
	}

	// Define a new Pod object
	pod := newPodForCR(instance)

	// Set {{ .Resource.Kind }} instance as the owner and controller
	if err := controllerutil.SetControllerReference(instance, pod, r.scheme); err != nil {
		return reconcile.Result{}, err
	}

	// Check if this Pod already exists
	found := &corev1.Pod{}
	err = r.client.Get(context.TODO(), types.NamespacedName{Name: pod.Name, Namespace: pod.Namespace}, found)
	if err != nil && errors.IsNotFound(err) {
		reqLogger.Info("Creating a new Pod", "Pod.Namespace", pod.Namespace, "Pod.Name", pod.Name)
		err = r.client.Create(context.TODO(), pod)
		if err != nil {
			return reconcile.Result{}, err
		}

		// Pod created successfully - don't requeue
		return reconcile.Result{}, nil
	} else if err != nil {
		return reconcile.Result{}, err
	}

	// Pod already exists - don't requeue
	reqLogger.Info("Skip reconcile: Pod already exists", "Pod.Namespace", found.Namespace, "Pod.Name", found.Name)
	return reconcile.Result{}, nil
}

// newPodForCR returns a busybox pod with the same name/namespace as the cr
func newPodForCR(cr *{{ .GoImportIdent }}.{{ .Resource.Kind }}) *corev1.Pod {
	labels := map[string]string{
		"app": cr.Name,
	}
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      cr.Name + "-pod",
			Namespace: cr.Namespace,
			Labels:    labels,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "busybox",
					Image:   "busybox",
					Command: []string{"sleep", "3600"},
				},
			},
		},
	}
}
`
