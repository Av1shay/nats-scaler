/*
Copyright 2025.

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

package controller

import (
	"context"
	"errors"
	"fmt"
	"github.com/Av1shay/nats-scaler/internal/errs"
	"github.com/Av1shay/nats-scaler/internal/nats"
	testutils "github.com/Av1shay/nats-scaler/test/utils"
	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	k8serrs "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"net/http"
	"net/http/httptest"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
	"time"

	scalingv1 "github.com/Av1shay/nats-scaler/api/v1"
)

var _ = Describe("ScalingRule Controller", func() {
	Context("When reconciling a resource", func() {
		const nsName = "default"
		const depName = "test-deployment"

		ctx := context.Background()

		It("should successfully reconcile the resource", func() {
			By("Create the necessary resources")

			natsServer := &mockNatsServer{
				resp: nats.JszResponse{
					AccountDetails: []nats.AccountDetails{{
						StreamDetail: []nats.StreamDetail{
							{Name: "ORDERS", ConsumerDetail: []nats.ConsumerDetail{{
								Name:       "orders-consumer",
								NumPending: 2,
							}}},
						},
					}},
				},
				statusCode:   200,
				gomega:       NewWithT(GinkgoT()),
				expectedPath: "/jsz",
			}
			ts := httptest.NewServer(natsServer)
			DeferCleanup(ts.Close)

			natsService := nats.NewService(&http.Client{Timeout: 5 * time.Second})
			mockedScaler := &mockScaler{}

			resourceName := fmt.Sprintf("test-resource-%d", GinkgoParallelProcess())
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: nsName,
			}
			spec := defaultNatsSpecs(depName, nsName, ts.URL)
			createDummyScalingRuleSpec(typeNamespacedName, spec)
			DeferCleanup(func() {
				resource := &scalingv1.ScalingRule{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				By("Cleanup the specific resource instance ScalingRule")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			})

			By("Reconciling the created resource")
			controllerReconciler := &ScalingRuleReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				NatsService: natsService,
				Scaler:      mockedScaler,
			}

			res, err := controllerReconciler.Reconcile(ctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})
			By("Asserting the reconcile return correct results")
			Expect(err).NotTo(HaveOccurred())
			Expect(res.RequeueAfter).To(Equal(time.Duration(spec.PollIntervalSeconds) * time.Second))

			By("Asserting the scaler was called with the expected parameters")
			Expect(mockedScaler.calledWith.Nn).To(Equal(types.NamespacedName{
				Name:      spec.DeploymentName,
				Namespace: spec.Namespace,
			}))
			Expect(mockedScaler.calledWith.Spec.MinReplicas).To(Equal(spec.MinReplicas))
			Expect(mockedScaler.calledWith.Spec.MaxReplicas).To(Equal(spec.MaxReplicas))
			Expect(mockedScaler.calledWith.Pending).To(Equal(2))
		})

		It("should produce correct error on NATS unexpected status code", func() {
			By("Create the necessary resources")

			natsServer := &mockNatsServer{
				statusCode:   500,
				gomega:       NewWithT(GinkgoT()),
				expectedPath: "/jsz",
			}
			ts := httptest.NewServer(natsServer)
			DeferCleanup(ts.Close)

			natsService := nats.NewService(&http.Client{Timeout: 5 * time.Second})
			mockedScaler := &mockScaler{}

			resourceName := fmt.Sprintf("test-resource-%d", GinkgoParallelProcess())
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: nsName,
			}
			spec := defaultNatsSpecs(depName, nsName, ts.URL)
			createDummyScalingRuleSpec(typeNamespacedName, spec)
			DeferCleanup(func() {
				resource := &scalingv1.ScalingRule{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				By("Cleanup the specific resource instance ScalingRule")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			})

			By("Reconciling the created resource")
			controllerReconciler := &ScalingRuleReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				NatsService: natsService,
				Scaler:      mockedScaler,
			}

			logger := &testutils.FakeLogger{}
			nctx := logr.NewContext(ctx, logr.New(logger))

			res, err := controllerReconciler.Reconcile(nctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			By("Asserting the reconcile return correct results")
			Expect(err).NotTo(HaveOccurred())
			Expect(logger.Buff.String()).To(ContainSubstring("failed to get pending messages from NATS"))
			Expect(res.RequeueAfter).To(Equal(errRequeueIntervalShort))

			Expect(logger.Err).To(HaveOccurred())
			var httpErr *errs.HTTPStatusCodeErr
			Expect(errors.As(logger.Err, &httpErr)).To(BeTrue(), "expected error to be of type *HTTPStatusCodeErr")
			Expect(httpErr.Code).To(Equal(500))
			Expect(string(httpErr.Body)).To(Equal("{\"account_details\":null}\n"))
		})

		It("should produce correct error on scaling error", func() {
			By("Create the necessary resources")

			natsServer := &mockNatsServer{
				resp: nats.JszResponse{
					AccountDetails: []nats.AccountDetails{{
						StreamDetail: []nats.StreamDetail{
							{Name: "ORDERS", ConsumerDetail: []nats.ConsumerDetail{{
								Name:       "orders-consumer",
								NumPending: 15,
							}}},
						},
					}},
				},
				statusCode:   200,
				gomega:       NewWithT(GinkgoT()),
				expectedPath: "/jsz",
			}
			ts := httptest.NewServer(natsServer)
			DeferCleanup(ts.Close)

			natsService := nats.NewService(&http.Client{Timeout: 5 * time.Second})

			expectedErr := errors.New("some error")
			mockedScaler := &mockScaler{err: expectedErr}

			resourceName := fmt.Sprintf("test-resource-%d", GinkgoParallelProcess())
			typeNamespacedName := types.NamespacedName{
				Name:      resourceName,
				Namespace: nsName,
			}
			spec := defaultNatsSpecs(depName, nsName, ts.URL)
			createDummyScalingRuleSpec(typeNamespacedName, spec)
			DeferCleanup(func() {
				resource := &scalingv1.ScalingRule{}
				err := k8sClient.Get(ctx, typeNamespacedName, resource)
				Expect(err).NotTo(HaveOccurred())
				By("Cleanup the specific resource instance ScalingRule")
				Expect(k8sClient.Delete(ctx, resource)).To(Succeed())
			})

			By("Reconciling the created resource")
			controllerReconciler := &ScalingRuleReconciler{
				Client:      k8sClient,
				Scheme:      k8sClient.Scheme(),
				NatsService: natsService,
				Scaler:      mockedScaler,
			}

			logger := &testutils.FakeLogger{}
			nctx := logr.NewContext(ctx, logr.New(logger))

			_, _ = controllerReconciler.Reconcile(nctx, reconcile.Request{
				NamespacedName: typeNamespacedName,
			})

			By("Asserting the scaler return expected error")
			Expect(errors.Is(logger.Err, expectedErr)).To(BeTrue())

			By("Asserting the scaler was called with the expected parameters")
			Expect(mockedScaler.calledWith.Nn).To(Equal(types.NamespacedName{
				Name:      spec.DeploymentName,
				Namespace: spec.Namespace,
			}))
			Expect(mockedScaler.calledWith.Spec.MinReplicas).To(Equal(spec.MinReplicas))
			Expect(mockedScaler.calledWith.Spec.MaxReplicas).To(Equal(spec.MaxReplicas))
			Expect(mockedScaler.calledWith.Pending).To(Equal(15))
		})
	})
})

func defaultNatsSpecs(depName, ns, natsURL string) scalingv1.ScalingRuleSpec {
	return scalingv1.ScalingRuleSpec{
		DeploymentName:      depName,
		Namespace:           ns,
		MinReplicas:         1,
		MaxReplicas:         3,
		NatsMonitoringURL:   natsURL,
		StreamName:          "ORDERS",
		ConsumerName:        "orders-consumer",
		ScaleUpThreshold:    10,
		ScaleDownThreshold:  2,
		PollIntervalSeconds: 10,
	}
}

func createDummyScalingRuleSpec(tns types.NamespacedName, spec scalingv1.ScalingRuleSpec) {
	By("creating the custom resource for the Kind ScalingRule")
	err := k8sClient.Get(ctx, tns, &scalingv1.ScalingRule{})
	if err != nil && k8serrs.IsNotFound(err) {
		resource := &scalingv1.ScalingRule{
			ObjectMeta: metav1.ObjectMeta{
				Name:      tns.Name,
				Namespace: tns.Namespace,
			},
			Spec: spec,
		}
		Expect(k8sClient.Create(ctx, resource)).To(Succeed())
	}
}
