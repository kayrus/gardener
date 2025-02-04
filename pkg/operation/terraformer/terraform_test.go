// Copyright (c) 2018 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package terraformer

import (
	"context"
	"encoding/json"
	"fmt"

	mockclient "github.com/gardener/gardener/pkg/mock/controller-runtime/client"
	kutil "github.com/gardener/gardener/pkg/utils/kubernetes"

	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var (
	configMapGroupResource = schema.GroupResource{Resource: "ConfigMaps"}
	secretGroupResource    = schema.GroupResource{Resource: "Secrets"}
)

var _ = Describe("terraformer", func() {
	var (
		ctrl *gomock.Controller
		c    *mockclient.MockClient
	)

	BeforeEach(func() {
		ctrl = gomock.NewController(GinkgoT())
		c = mockclient.NewMockClient(ctrl)
	})

	AfterEach(func() {
		ctrl.Finish()
	})

	Describe("#CreateOrUpdateConfigurationConfigMap", func() {
		It("Should create the config map", func() {
			const (
				namespace = "namespace"
				name      = "name"

				main      = "main"
				variables = "variables"
			)

			var (
				ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: name}
				expected   = &corev1.ConfigMap{
					ObjectMeta: ObjectMeta,
					Data: map[string]string{
						MainKey:      main,
						VariablesKey: variables,
					},
				}
			)

			gomock.InOrder(
				c.EXPECT().
					Get(gomock.Any(), kutil.Key(namespace, name), &corev1.ConfigMap{ObjectMeta: ObjectMeta}).
					Return(apierrors.NewNotFound(configMapGroupResource, name)),
				c.EXPECT().
					Create(gomock.Any(), expected.DeepCopy()),
			)

			actual, err := CreateOrUpdateConfigurationConfigMap(context.TODO(), c, namespace, name, main, variables)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		})
	})

	Describe("#CreateStateConfigMap", func() {
		It("Should create the config map", func() {
			const (
				namespace = "namespace"
				name      = "name"

				state = "state"
			)

			var (
				ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: name}
				expected   = &corev1.ConfigMap{
					ObjectMeta: ObjectMeta,
					Data: map[string]string{
						StateKey: state,
					},
				}
			)

			c.EXPECT().Create(gomock.Any(), expected.DeepCopy())

			err := CreateStateConfigMap(context.TODO(), c, namespace, name, state)
			Expect(err).NotTo(HaveOccurred())
		})
	})

	Describe("#CreateOrUpdateTFVarsSecret", func() {
		It("Should create the secret", func() {
			const (
				namespace = "namespace"
				name      = "name"
			)

			var (
				tfVars     = []byte("tfvars")
				ObjectMeta = metav1.ObjectMeta{Namespace: namespace, Name: name}
				expected   = &corev1.Secret{
					ObjectMeta: ObjectMeta,
					Data: map[string][]byte{
						TFVarsKey: tfVars,
					},
				}
			)

			gomock.InOrder(
				c.EXPECT().
					Get(gomock.Any(), kutil.Key(namespace, name), &corev1.Secret{ObjectMeta: ObjectMeta}).
					Return(apierrors.NewNotFound(secretGroupResource, name)),
				c.EXPECT().
					Create(gomock.Any(), expected.DeepCopy()),
			)

			actual, err := CreateOrUpdateTFVarsSecret(context.TODO(), c, namespace, name, tfVars)
			Expect(err).NotTo(HaveOccurred())
			Expect(actual).To(Equal(expected))
		})
	})

	Describe("#DefaultInitializer", func() {
		const (
			namespace         = "namespace"
			configurationName = "configuration"
			variablesName     = "variables"
			stateName         = "state"

			main      = "main"
			variables = "variables"
		)

		var (
			tfVars = []byte("tfvars")

			configurationKey = kutil.Key(namespace, configurationName)
			variablesKey     = kutil.Key(namespace, variablesName)

			configurationObjectMeta = kutil.ObjectMeta(namespace, configurationName)
			variablesObjectMeta     = kutil.ObjectMeta(namespace, variablesName)
			stateObjectMeta         = kutil.ObjectMeta(namespace, stateName)

			getConfiguration = &corev1.ConfigMap{ObjectMeta: configurationObjectMeta}
			getVariables     = &corev1.Secret{ObjectMeta: variablesObjectMeta}

			createConfiguration = &corev1.ConfigMap{
				ObjectMeta: configurationObjectMeta,
				Data: map[string]string{
					MainKey:      main,
					VariablesKey: variables,
				},
			}
			createVariables = &corev1.Secret{
				ObjectMeta: variablesObjectMeta,
				Data: map[string][]byte{
					TFVarsKey: tfVars,
				},
			}
			createState = &corev1.ConfigMap{
				ObjectMeta: stateObjectMeta,
				Data: map[string]string{
					StateKey: "",
				},
			}

			configurationNotFound = apierrors.NewNotFound(configMapGroupResource, configurationName)
			variablesNotFound     = apierrors.NewNotFound(secretGroupResource, variablesName)
		)

		runInitializer := func(initializeState bool) error {
			return DefaultInitializer(c, main, variables, tfVars)(&InitializerConfig{
				Namespace:         namespace,
				ConfigurationName: configurationName,
				VariablesName:     variablesName,
				StateName:         stateName,
				InitializeState:   initializeState,
			})
		}

		It("should create all resources", func() {
			gomock.InOrder(
				c.EXPECT().
					Get(gomock.Any(), configurationKey, getConfiguration.DeepCopy()).
					Return(configurationNotFound),
				c.EXPECT().
					Create(gomock.Any(), createConfiguration.DeepCopy()),

				c.EXPECT().
					Get(gomock.Any(), variablesKey, getVariables.DeepCopy()).
					Return(variablesNotFound),
				c.EXPECT().
					Create(gomock.Any(), createVariables.DeepCopy()),

				c.EXPECT().
					Create(gomock.Any(), createState.DeepCopy()),
			)

			Expect(runInitializer(true)).NotTo(HaveOccurred())
		})

		It("should not initialize state when initializeState is false", func() {
			gomock.InOrder(
				c.EXPECT().
					Get(gomock.Any(), configurationKey, getConfiguration.DeepCopy()).
					Return(configurationNotFound),
				c.EXPECT().
					Create(gomock.Any(), createConfiguration.DeepCopy()),

				c.EXPECT().
					Get(gomock.Any(), variablesKey, getVariables.DeepCopy()).
					Return(variablesNotFound),
				c.EXPECT().
					Create(gomock.Any(), createVariables.DeepCopy()),
			)

			Expect(runInitializer(false)).NotTo(HaveOccurred())
		})
	})

	Describe("#GetStateOutputVariables", func() {
		const (
			namespace = "namespace"
			name      = "name"
			purpose   = "purpose"
			image     = "image"
		)

		var (
			stateName = fmt.Sprintf("%s.%s.tf-state", name, purpose)
			stateKey  = kutil.Key(namespace, stateName)
		)

		It("should return err when state version is not supported", func() {
			state := map[string]interface{}{
				"version": 1,
			}
			stateJSON, err := json.Marshal(state)
			Expect(err).NotTo((HaveOccurred()))

			c.EXPECT().
				Get(gomock.Any(), stateKey, gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
					cm.Data = map[string]string{
						StateKey: string(stateJSON),
					}
					return nil
				})

			terraformer := New(nil, c, nil, purpose, namespace, name, image)
			actual, err := terraformer.GetStateOutputVariables("variableV1")

			Expect(actual).To(BeNil())
			Expect(err).To(HaveOccurred())
		})

		It("should get state v3 output variables", func() {
			state := map[string]interface{}{
				"version": 3,
				"modules": []map[string]interface{}{
					{
						"outputs": map[string]interface{}{
							"variableV3": map[string]string{
								"value": "valueV3",
							},
						},
					},
				},
			}
			stateJSON, err := json.Marshal(state)
			Expect(err).NotTo((HaveOccurred()))

			c.EXPECT().
				Get(gomock.Any(), stateKey, gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
					cm.Data = map[string]string{
						StateKey: string(stateJSON),
					}
					return nil
				})

			terraformer := New(nil, c, nil, purpose, namespace, name, image)
			actual, err := terraformer.GetStateOutputVariables("variableV3")

			expected := map[string]string{
				"variableV3": "valueV3",
			}
			Expect(actual).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		})

		It("should get state v4 output variables", func() {
			state := map[string]interface{}{
				"version": 4,
				"outputs": map[string]interface{}{
					"variableV4": map[string]string{
						"value": "valueV4",
					},
				},
			}
			stateJSON, err := json.Marshal(state)
			Expect(err).NotTo((HaveOccurred()))

			c.EXPECT().
				Get(gomock.Any(), stateKey, gomock.AssignableToTypeOf(&corev1.ConfigMap{})).
				DoAndReturn(func(_ context.Context, _ client.ObjectKey, cm *corev1.ConfigMap) error {
					cm.Data = map[string]string{
						StateKey: string(stateJSON),
					}
					return nil
				})

			terraformer := New(nil, c, nil, purpose, namespace, name, image)
			actual, err := terraformer.GetStateOutputVariables("variableV4")

			expected := map[string]string{
				"variableV4": "valueV4",
			}
			Expect(actual).To(Equal(expected))
			Expect(err).NotTo(HaveOccurred())
		})
	})
})
